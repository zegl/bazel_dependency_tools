package maven_jar

import (
	"encoding/xml"
	"errors"
	"fmt"
	"go.starlark.net/syntax"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/zegl/bazel_dependency_tools/internal"
)

type NewestVersionResolver func(coordinate string) (string, error)

type Meta struct {
	XMLName    xml.Name `xml:"metadata"`
	Versioning struct {
		XMLName xml.Name `xml:"versioning"`
		Latest  string   `xml:"latest"`
	} `xml:"versioning"`
}

func NewestAvailable(coordinate string) (string, error) {
	xyz := strings.Split(coordinate, ":")

	resp, err := http.Get(fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/maven-metadata.xml", strings.ReplaceAll(xyz[0], ".", "/"), xyz[1]))
	if err != nil {
		return "", fmt.Errorf("failed to fetch from repo1.maven.org: %w", err)
	}
	defer resp.Body.Close()

	allData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading XML response from maven failed: %w", err)
	}

	var meta Meta
	err = xml.Unmarshal(allData, &meta)
	if err != nil {
		return "", fmt.Errorf("unmarshal maven XML failed: %w", err)
	}

	return meta.Versioning.Latest, nil
}

func Check(e *syntax.CallExpr, versionFunc NewestVersionResolver) ([]internal.LineReplacement, error) {
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

	log.Printf("Checking %s", mavenJarName)
	_ = mavenJarSha1

	newestVersion, err := versionFunc(mavenJarArtifact.Value.(string))
	if err != nil {
		return nil, fmt.Errorf("unable to find newer maven_jar: %w", err)
	}

	xyz := strings.Split(mavenJarArtifact.Value.(string), ":")
	xyz[2] = newestVersion

	replacements = append(replacements, internal.LineReplacement{
		Filename:     mavenJarArtifact.TokenPos.Filename(),
		Line:         mavenJarArtifact.TokenPos.Line,
		Find:         mavenJarArtifact.Value.(string),
		Substitution: strings.Join(xyz, ":"),
	})

	if len(replacements) != 0 {
		return replacements, nil
	}

	return nil, errors.New("no match")
}
