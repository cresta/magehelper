package env

import "os"

type Env struct {
	getEnv  func(string) string
	environ func() []string
}

func New(getEnv func(string) string, environ func() []string) *Env {
	return &Env{
		getEnv:  getEnv,
		environ: environ,
	}
}

func (e *Env) osGetEnv(s string) string {
	if e == nil || e.getEnv == nil {
		return os.Getenv(s)
	}
	return e.getEnv(s)
}

func (e *Env) osEnviron() []string {
	if e == nil || e.getEnv == nil {
		return os.Environ()
	}
	return e.environ()
}

func (e *Env) AddEnv(env ...string) []string {
	r := make([]string, 0, len(e.osEnviron()))
	r = append(r, e.osEnviron()...)
	r = append(r, env...)
	return r
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
