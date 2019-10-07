package internal

type LineReplacement struct {
	Filename           string
	Line               int32
	Find, Substitution string
}
