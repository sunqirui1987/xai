/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package seedance

import (
	"context"
	"strings"

	xai "github.com/goplus/xai/spec"
)

type serviceProvider interface {
	SeedanceService() *Service
}

func (p *Service) Actions(model xai.Model) []xai.Action {
	if IsVideoModel(string(model)) {
		return []xai.Action{xai.GenVideo}
	}
	return nil
}

func (p *Service) Operation(model xai.Model, action xai.Action) (xai.Operation, error) {
	if action != xai.GenVideo || !IsVideoModel(string(model)) {
		return nil, xai.ErrNotFound
	}
	return &genVideo{model: strings.TrimSpace(string(model))}, nil
}

type genVideo struct {
	model  string
	params *Params
}

func (p *genVideo) InputSchema() xai.InputSchema {
	return &inputSchema{}
}

func (p *genVideo) Params() xai.Params {
	if p.params == nil {
		p.params = NewParams()
	}
	return p.params
}

func (p *genVideo) Call(ctx context.Context, svc xai.Service, opts xai.OptionBuilder) (xai.OperationResponse, error) {
	s, ok := svc.(serviceProvider)
	if !ok {
		return nil, xai.ErrNotFound
	}
	sd := s.SeedanceService()
	if sd == nil || sd.backend == nil {
		return nil, xai.ErrNotFound
	}

	params := p.Params().(*Params)
	if params.PrimaryText() == "" {
		return nil, ErrTextRequired
	}

	schema := p.InputSchema()
	if err := validateParamsAgainstRestriction(schema, params); err != nil {
		return nil, err
	}

	_ = opts // Seedance Options currently unused
	return sd.backend.Submit(ctx, xai.Model(p.model), params)
}

type inputSchema struct{}

func (s *inputSchema) Fields() []xai.Field { return GenVideoFields() }

func (s *inputSchema) Restrict(name string) *xai.Restriction {
	return GenVideoRestrict(name)
}

func validateParamsAgainstRestriction(schema xai.InputSchema, params *Params) error {
	for name, val := range params.Export() {
		if strings.HasPrefix(name, "_") {
			continue
		}
		s, ok := val.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if r := schema.Restrict(name); r != nil {
			if err := r.ValidateString(name, s); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetTask resumes polling for an existing task id.
func (p *Service) GetTask(ctx context.Context, model xai.Model, action xai.Action, taskID string) (xai.OperationResponse, error) {
	if action != xai.GenVideo || p.backend == nil {
		return nil, xai.ErrNotFound
	}
	if strings.TrimSpace(taskID) == "" {
		return nil, xai.ErrNotFound
	}
	return p.backend.GetTaskStatus(ctx, strings.TrimSpace(taskID))
}
