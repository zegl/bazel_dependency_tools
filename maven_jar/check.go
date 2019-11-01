package maven_jar

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/blang/semver"
	"go.starlark.net/syntax"

	"github.com/zegl/bazel_dependency_tools/internal"
)

type NewestVersionResolver func(coordinate string) (version, sha1 string, err error)

type Meta struct {
	XMLName    xml.Name `xml:"metadata"`
	Versioning struct {
		XMLName  xml.Name `xml:"versioning"`
		Versions []struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
}

func NewestAvailable(coordinate string) (string, string, error) {
	xyz := strings.Split(coordinate, ":")

	resp, err := http.Get(fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/maven-metadata.xml", strings.ReplaceAll(xyz[0], ".", "/"), xyz[1]))
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch from repo1.maven.org: %w", err)
	}
	defer resp.Body.Close()

	allData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading XML response from maven failed: %w", err)
	}

	var meta Meta
	err = xml.Unmarshal(allData, &meta)
	if err != nil {
		return "", "", fmt.Errorf("unmarshal maven XML failed: %w", err)
	}

	// Find the newest version available
	var newestVersion semver.Version

	for _, versions := range meta.Versioning.Versions {
		for _, version := range versions.Version {
			if v, err := semver.Parse(version); err == nil {
				if v.GT(newestVersion) {
					newestVersion = v
				}
			}
		}
	}

	sha1, err := mavenCentralSha1(xyz[0], xyz[1], newestVersion.String())
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch sha1: %w", err)
	}

	return newestVersion.String(), sha1, nil
}

func mavenCentralSha1(x, y, z string) (string, error) {
	// Example: https://repo1.maven.org/maven2/io/opencensus/opencensus-api/0.24.0/opencensus-api-0.24.0.jar.sha1
	resp, err := http.Get(fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.jar.sha1", strings.ReplaceAll(x, ".", "/"), y, z, y, z))
	if err != nil {
		return "", fmt.Errorf("failed to fetch sha1 from repo1.maven.org: %w", err)
	}
	defer resp.Body.Close()

	allData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read sha1: %w", err)
	}

	// Some .jar.sha1 files contains something like this:
	// "df853af9fe34d4eb6f849a1b5936fddfcbe67751  /home/projects/maven/repository-staging/to-ibiblio/maven2/mx4j/mx4j-tools/3.0.1/mx4j-tools-3.0.1.jar"
	sha1 := strings.Split(strings.TrimSpace(string(allData)), " ")[0]
	if len(sha1) != 40 {
		return "", errors.New("unexpected length")
	}
	return sha1, nil
}

func Check(e *syntax.CallExpr, namePrefixFilter string, versionFunc NewestVersionResolver) ([]internal.LineReplacement, error) {
	var replacements []internal.LineReplacement

	var mavenJarName string
	var mavenJarArtifact *syntax.Literal
	var mavenJarSha1 *syntax.Literal
	// var mavenJarSha256 *syntax.Literal

	for _, arg := range e.Args {
		if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
			if xIdent, ok := binExp.X.(*syntax.Ident); ok {
				switch xIdent.Name {
				case "name":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						mavenJarName = rhs.Value.(string)
					}
				case "artifact":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						mavenJarArtifact = rhs
					}
				case "sha1":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						mavenJarSha1 = rhs
					}
				}
			}
		}
	}

	// Don't attempt to upgrade this dependency
	if !strings.HasPrefix(mavenJarName, namePrefixFilter) {
		return nil, nil
	}

	if mavenJarArtifact == nil {
		return nil, fmt.Errorf("unable to parse %s", mavenJarName)
	}

	log.Printf("Checking %s", mavenJarName)
	_ = mavenJarSha1

	newestVersion, sha1, err := versionFunc(mavenJarArtifact.Value.(string))
	if err != nil {
		return nil, fmt.Errorf("unable to find newer maven_jar: %w", err)
	}

	xyz := strings.Split(mavenJarArtifact.Value.(string), ":")

	// No newer version found
	if xyz[2] == newestVersion {
		return nil, nil
	}

	newXyz := xyz
	newXyz[2] = newestVersion

	log.Printf("Found: version=%s sha1=%s", newestVersion, sha1)

	replacements = append(replacements, internal.LineReplacement{
		Filename:     mavenJarArtifact.TokenPos.Filename(),
		Line:         mavenJarArtifact.TokenPos.Line,
		Find:         mavenJarArtifact.Value.(string),
		Substitution: strings.Join(newXyz, ":"),
	})

	if mavenJarSha1 != nil {
		replacements = append(replacements, internal.LineReplacement{
			Filename:     mavenJarSha1.TokenPos.Filename(),
			Line:         mavenJarSha1.TokenPos.Line,
			Find:         mavenJarSha1.Value.(string),
			Substitution: sha1,
		})
	}

	if len(replacements) != 0 {
		return replacements, nil
	}

	return nil, errors.New("no match")
}
