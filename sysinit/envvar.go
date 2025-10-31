// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

// EnvVars is a map of environment variable values by name.
type EnvVars map[string]string

// SetEnv sets the given [EnvVars] in the environment.
func SetEnv(envVars EnvVars) error {
	for key, value := range envVars {
		err := setenv(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// WithEnv returns a setup [Func] that wraps [SetEnv] and can be used with
// [Run].
func WithEnv(envVars EnvVars) Func {
	return func(_ *State) error {
		return SetEnv(envVars)
	}
}
