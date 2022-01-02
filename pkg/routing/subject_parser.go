package routing

import (
	"errors"
	"strings"
)

const (
	ValidSubjectElements     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRTSUVWXYZ0123456789"
	SubjectDelimiter         = "."
	SubjectSingleWildcard    = "*"
	SubjectMultipleWildcards = ">"
)

var (
	InvalidSubjectError = errors.New("invalid subject")
)

func MatchSubject(subject string, subscription string) bool {
	subjectChunks := strings.Split(subject, SubjectDelimiter)
	subscriptionChunks := strings.Split(subscription, SubjectDelimiter)

	for index, c := range subjectChunks {
		if subscriptionChunks[index] == ">" {
			return true
		}

		if len(subscriptionChunks) < len(subjectChunks) {
			return false
		}

		if subscriptionChunks[index] == "*" {
			continue
		}

		if c != subscriptionChunks[index] {
			return false
		}
	}

	return true
}

func ValidateSubject(subject string) error {
	chunks := strings.Split(subject, SubjectDelimiter)

	for index, chunk := range chunks {
		foundWildcard, err := checkForWildcards(index, chunk, chunks)
		if err != nil {
			return err
		}
		// If wildcards have been found, skip processing the current chunk
		if foundWildcard {
			continue
		}
		validChars := checkIfChunkHasValidCharacters(chunk)
		if !validChars {
			return InvalidSubjectError
		}
	}
	return nil
}

func checkIfChunkHasValidCharacters(chunk string) bool {
	for _, ch := range chunk {
		// Checks digits, upper case letters, and lower case letters
		if (ch >= 48 && ch <= 57) || (ch >= 65 && ch <= 90) || (ch >= 97 && ch <= 122) {

		} else {
			return false
		}
	}
	return true
}

func checkForWildcards(index int, chunk string, chunks []string) (bool, error) {
	if len(chunk) == 1 {
		if strings.EqualFold(chunk, SubjectMultipleWildcards) {
			// Check if there are more chunks
			if len(chunks) > index+1 {
				return true, InvalidSubjectError
			}
			return true, nil
		}

		if strings.EqualFold(chunk, SubjectSingleWildcard) {
			return true, nil
		}
	}
	return false, nil
}
