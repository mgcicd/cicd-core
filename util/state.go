package util

type YesOrNo int

const (
	Yes YesOrNo = iota
	No          = -1
)

type TrueOrFalse int

const (
	False TrueOrFalse = iota
	True  TrueOrFalse = 1
)
