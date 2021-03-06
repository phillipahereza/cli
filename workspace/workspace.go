package workspace

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var errMissingMetadata = errors.New("no solution metadata file found")

// IsMissingMetadata verifies the type of error.
func IsMissingMetadata(err error) bool {
	return err == errMissingMetadata
}

// Workspace represents a user's Exercism workspace.
// It may contain a user's own exercises, and other people's
// exercises that they've downloaded to look at or run locally.
type Workspace struct {
	Dir string
}

// New returns a configured workspace.
func New(dir string) (Workspace, error) {
	_, err := os.Lstat(dir)
	if err != nil {
		return Workspace{}, err
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return Workspace{}, err
	}
	return Workspace{Dir: dir}, nil
}

// PotentialExercises are a first-level guess at the user's exercises.
// It looks at the workspace structurally, and guesses based on
// the location of the directory. E.g. any top level directory
// within the workspace (except 'users') is assumed to be a
// track, and any directory within there again is assumed to
// be an exercise.
func (ws Workspace) PotentialExercises() ([]Exercise, error) {
	exercises := []Exercise{}

	topInfos, err := ioutil.ReadDir(ws.Dir)
	if err != nil {
		return nil, err
	}
	for _, topInfo := range topInfos {
		if !topInfo.IsDir() {
			continue
		}

		if topInfo.Name() == "users" {
			continue
		}

		subInfos, err := ioutil.ReadDir(filepath.Join(ws.Dir, topInfo.Name()))
		if err != nil {
			return nil, err
		}

		for _, subInfo := range subInfos {
			if !subInfo.IsDir() {
				continue
			}

			exercises = append(exercises, Exercise{Track: topInfo.Name(), Slug: subInfo.Name(), Root: ws.Dir})
		}
	}

	return exercises, nil
}

// Exercises returns the user's exercises within the workspace.
// This doesn't find legacy exercises where the metadata is missing.
func (ws Workspace) Exercises() ([]Exercise, error) {
	candidates, err := ws.PotentialExercises()
	if err != nil {
		return nil, err
	}

	exercises := make([]Exercise, 0, len(candidates))
	for _, candidate := range candidates {
		ok, err := candidate.HasMetadata()
		if err != nil {
			return nil, err
		}
		if ok {
			exercises = append(exercises, candidate)
		}
	}
	return exercises, nil
}

// SolutionDir determines the root directory of a solution.
// This is the directory that contains the solution metadata file.
func (ws Workspace) SolutionDir(s string) (string, error) {
	if !strings.HasPrefix(s, ws.Dir) {
		return "", errors.New("not in workspace")
	}

	path := s
	for {
		if path == ws.Dir {
			return "", errMissingMetadata
		}
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			return "", err
		}
		if _, err := os.Lstat(filepath.Join(path, metadataFilepath)); err == nil {
			return path, nil
		}
		if _, err := os.Lstat(filepath.Join(path, legacySolutionFilename)); err == nil {
			return path, nil
		}
		path = filepath.Dir(path)
	}
}
