// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "github.com/spf13/viper"

// Option allows custom hooks before and after config loading.
type Option struct {
	pre  func(v *viper.Viper, cfg *Config) error
	post func(v *viper.Viper, cfg *Config) error
}

func (o Option) applyPre(v *viper.Viper, cfg *Config) error {
	if o.pre != nil {
		return o.pre(v, cfg)
	}
	return nil
}
func (o Option) applyPost(v *viper.Viper, cfg *Config) error {
	if o.post != nil {
		return o.post(v, cfg)
	}
	return nil
}

// WithDefaults overrides default values before the config file is read.
func WithDefaults(apply func(cfg *Config)) Option {
	return Option{
		pre: func(_ *viper.Viper, cfg *Config) error {
			if apply != nil {
				apply(cfg)
			}
			return nil
		},
	}
}

// WithViper customizes viper before config loading, including search paths, aliases, and env-key replacement.
func WithViper(apply func(v *viper.Viper)) Option {
	return Option{
		pre: func(v *viper.Viper, _ *Config) error {
			if apply != nil && v != nil {
				apply(v)
			}
			return nil
		},
	}
}

// WithEnvPrefix sets the environment variable prefix, defaulting to CHOYSUM.
func WithEnvPrefix(prefix string) Option {
	return WithViper(func(v *viper.Viper) {
		if prefix != "" {
			v.SetEnvPrefix(prefix)
		}
	})
}

// UnmarshalKey unmarshals the named section into a custom struct after config loading.
func UnmarshalKey[T any](key string, out *T) Option {
	return Option{
		post: func(v *viper.Viper, _ *Config) error {
			if key == "" || out == nil {
				return nil
			}
			return v.UnmarshalKey(key, out)
		},
	}
}

// AfterUnmarshal runs custom logic after config unmarshalling, such as validation or derived fields.
func AfterUnmarshal(apply func(v *viper.Viper, cfg *Config) error) Option {
	return Option{
		post: apply,
	}
}
