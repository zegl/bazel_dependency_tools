package maven_jar

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"go.starlark.net/syntax"
)

type LIC string

const (
	Apache20Grep LIC = "Apache-2.0"
)

var ErrSkipped = errors.New("skipped")

func License(e *syntax.CallExpr, namePrefixFilter string) (string, LIC, error) {
	var mavenJarName string
	var mavenJarArtifact *syntax.Literal
	repository := "https://repo1.maven.org/maven2"

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
				case "repository":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						repository = rhs.Value.(string)
					}
				}
			}
		}
	}

	// Don't check this dependency
	if !strings.HasPrefix(mavenJarName, namePrefixFilter) {
		return "", "", ErrSkipped
	}

	if mavenJarArtifact == nil {
		return mavenJarName, "", fmt.Errorf("unable to parse %s", mavenJarName)
	}

	x, y, z := strToCoord(mavenJarArtifact.Value.(string))
	license, err := mavenLicense(repository, x, y, z)
	if err != nil {
		return mavenJarName, "", err
	}
	return mavenJarName, license, nil
}

func strToCoord(s string) (string, string, string) {
	xyz := strings.Split(s, ":")
	return xyz[0], xyz[1], xyz[len(xyz)-1]
}

type ArtifactLicense struct {
	Art     string
	License LIC
}

func LicenseMavenInstall(e *syntax.CallExpr, namePrefixFilter, workspacePath string) ([]ArtifactLicense, error) {
	var mavenInstallName string
	var pinningJson string
	// var mavenJarArtifact *syntax.Literal
	// repository := "https://repo1.maven.org/maven2"
	for _, arg := range e.Args {
		if binExp, ok := arg.(*syntax.BinaryExpr); ok && binExp.Op == syntax.EQ {
			if xIdent, ok := binExp.X.(*syntax.Ident); ok {
				switch xIdent.Name {
				case "name":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						mavenInstallName = rhs.Value.(string)
					}
				case "maven_install_json":
					if rhs, ok := binExp.Y.(*syntax.Literal); ok {
						pinningJson = rhs.Value.(string)
					}
				}
			}
		}
	}

	// Don't check this dependency
	if !strings.HasPrefix(mavenInstallName, namePrefixFilter) {
		return nil, ErrSkipped
	}

	p := path.Join(path.Dir(workspacePath), strings.ReplaceAll(pinningJson, ":", "/"))

	pinningJsonData, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	type pinningSchema struct {
		DependencyTree struct {
			Dependencies []struct {
				Coord              string        `json:"coord"`
				Dependencies       []interface{} `json:"dependencies"`
				DirectDependencies []interface{} `json:"directDependencies"`
				File               string        `json:"file"`
				MirrorUrls         []string      `json:"mirror_urls"`
				Sha256             string        `json:"sha256"`
				URL                string        `json:"url"`
			} `json:"dependencies"`
			Version string `json:"version"`
		} `json:"dependency_tree"`
	}

	var pinning pinningSchema
	err = json.Unmarshal(pinningJsonData, &pinning)
	if err != nil {
		return nil, err
	}

	var res []ArtifactLicense

	for _, dep := range pinning.DependencyTree.Dependencies {
		x, y, z := strToCoord(dep.Coord)
		license, err := mavenLicense("https://repo1.maven.org/maven2", x, y, z)
		if err != nil {
			return nil, err
		}
		res = append(res, ArtifactLicense{
			Art:     dep.Coord,
			License: license,
		})
	}

	return res, nil
}

func mavenLicense(repository, x, y, z string) (LIC, error) {
	pom, err := fetchPom(repository, x, y, z)
	if err != nil {
		return "", err
	}

	if strings.Contains(pom, "http://www.apache.org/licenses/LICENSE-2.0") {
		return "Apache License, Version 2.0", nil
	}
	if strings.Contains(pom, "Apache License, Version 2.0") {
		return "Apache License, Version 2.0", nil
	}
	if strings.Contains(pom, "Eclipse Public License v1.0") {
		return "Eclipse Public License v1.0", nil
	}
	if strings.Contains(pom, "GNU Lesser General Public License version 2.1") {
		return "GNU Lesser General Public License version 2.1", nil
	}

	type pomxml struct {
		Licenses []struct {
			License struct {
				Name string `xml:"name"`
			} `xml:"license"`
		} `xml:"licenses"`

		Parent struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Version    string `xml:"version"`
		} `xml:"parent"`
	}

	var l pomxml
	err = xml.Unmarshal([]byte(pom), &l)
	if err != nil {
		return "", fmt.Errorf("unmarshal maven XML failed: %w", err)
	}

	if len(l.Licenses) == 1 {
		return LIC(l.Licenses[0].License.Name), nil
	}

	// If has parent, check there
	if l.Parent.ArtifactID != "" {
		return mavenLicense(repository, l.Parent.GroupID, l.Parent.ArtifactID, l.Parent.Version)
	}

	// Check newer version
	newZ, _, err := NewestAvailable(fmt.Sprintf("%s:%s:%s", x, y, z), nil)
	if newZ != z && err == nil {
		if l, err := mavenLicense(repository, x, y, newZ); err == nil {
			return l, nil
		}
	}

	return "", errors.New("no license found")
}

func fetchPom(repository, x, y, z string) (string, error) {
	// https://repo1.maven.org/maven2/net/sourceforge/argparse4j/argparse4j/0.4.3/argparse4j-0.4.3.pom
	// https://repo1.maven.org/maven2/software/amazon/awssdk/aws-query-protocol/2.7.5/aws-query-protocol-2.7.5.pom
	// https://repo1.maven.org/maven2/software/amazon/awssdk/aws-xml-protocol/jar/aws-xml-protocol-jar.pom
	url := fmt.Sprintf("%s/%s/%s/%s/%s-%s.pom", repository, strings.ReplaceAll(x, ".", "/"), y, z, y, z)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sha1 from repo1.maven.org: %w", err)
	}
	defer resp.Body.Close()

	allData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read sha1: %w", err)
	}

	return string(allData), nil
}
