package gopty

import (
	"os"
	"strings"
)

type Env struct {
	Key   string
	Value string
}

func NewEnv(key, value string) Env {
	return Env{
		Key:   strings.TrimSpace(key),
		Value: strings.TrimSpace(value),
	}
}

func (e Env) Environ() string {
	return e.Key + "=" + e.Value
}

func (e Env) Expand(expanded map[string]string) string {
	result := e.Value
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varName := result[start+2 : end]
		replacement, ok := expanded[varName]
		if !ok {
			replacement = os.Getenv(varName)
		}
		result = result[:start] + replacement + result[end+1:]
	}
	return result
}

func ExpandAll(envs []Env) []Env {
	expanded := make(map[string]string)
	result := make([]Env, len(envs))
	for i, e := range envs {
		e.Value = e.Expand(expanded)
		expanded[e.Key] = e.Value
		result[i] = e
	}
	return result
}
