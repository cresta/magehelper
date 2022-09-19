package env

import (
	"fmt"
	"os"
)

type Env struct {
	getEnv  func(string) string
	environ func() []string
}

// Default will set a default key/value inside the environment only if value is empty string.  It returns a function
// you can optionally later call to set the environment back to the original, unset value.
func Default(key string, value string) func() {
	if _, exists := os.LookupEnv(key); !exists {
		if err := os.Setenv(key, value); err != nil {
			fmt.Printf("unable to set environment variable %s = %s (%s)\n", key, value, err)
			return func() {}
		}
		return func() {
			if err := os.Unsetenv(key); err != nil {
				fmt.Printf("Unable to restore default environment: %s\n", err)
			}
		}
	}
	return func() {}
}

var Instance = &Env{}

func New(getEnv func(string) string, environ func() []string) *Env {
	return &Env{
		getEnv:  getEnv,
		environ: environ,
	}
}

func NewFromMap(e map[string]string) *Env {
	return &Env{
		getEnv: func(s string) string {
			return e[s]
		},
		environ: func() []string {
			r := make([]string, 0, len(e))
			for k, v := range e {
				r = append(r, k+"="+v)
			}
			return r
		},
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
