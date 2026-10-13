package onboarding

type IDGenerator interface {
	Generate() (string, error)
}
