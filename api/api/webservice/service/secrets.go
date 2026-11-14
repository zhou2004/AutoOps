package service

import "dodevops-api/pkg/websecret"

func seal(v string) (string, error)   { return websecret.Seal(v) }
func unseal(v string) (string, error) { return websecret.Unseal(v) }
