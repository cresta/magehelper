package env

import "os"

type Env struct {
	getEnv func(string) string
}

func New(getEnv func(string) string) *Env {
	return &Env{
		getEnv: getEnv,
	}
}

func (e *Env) osGetEnv(s string) string {
	if e == nil || e.getEnv == nil {
		return os.Getenv(s)
	}
	return e.getEnv(s)
}

func (e *Env) GetDefault(env string, def string) string {
	if s := e.osGetEnv(env); s != "" {
		return s
	}
	return def
}

func (e *Env) Get(env string) string {
	return e.GetDefault(env, "")
}
