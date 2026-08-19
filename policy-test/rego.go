// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/topdown/print"
)

type OPAPolicy struct {
	compiler *ast.Compiler
	store    storage.Store
}

var _ Policy = &OPAPolicy{}

func NewOPAPolicy(policy string) (*OPAPolicy, error) {
	compiler := ast.NewCompiler().WithEnablePrintStatements(true)
	module, err := ast.ParseModule("policy.rego", policy)
	if err != nil {
		return nil, fmt.Errorf("parsing policy module: %w", err)
	}
	compiler.Compile(map[string]*ast.Module{"policy.rego": module})
	if compiler.Failed() {
		return nil, fmt.Errorf("compiling policy module: %w", compiler.Errors)
	}

	return &OPAPolicy{
		compiler: compiler,
		// We need to initialize the store with an empty "pstate" object so that the policy can read from it.
		store: inmem.NewFromObject(map[string]any{"pstate": map[string]any{}}),
	}, nil
}

// collectingPrintHook is a hook that collects all print statements from the rego evaluation.
type collectingPrintHook struct {
	messages []string
}

func (p *collectingPrintHook) Print(_ print.Context, msg string) error {
	p.messages = append(p.messages, msg)
	return nil
}

// AllowRequest evaluates the given test case against the policy.
func (p *OPAPolicy) AllowRequest(ctx context.Context, tc TestCase) (bool, string, error) {
	req, ok := tc.Request.(map[string]any)
	if !ok {
		return false, "", fmt.Errorf("request is not a map[string]any, got: %T", tc.Request)
	}
	if _, isMetadata := req["ops"]; isMetadata {
		// The agent logs these responses during policy evaluation, but they are not the actual request. We can ignore them here.
		// TODO(davidweisse): Should we verify these against the returned ops from the policy evaluation?
		return true, "", nil
	}

	query := fmt.Sprintf("data.agent_policy.%s", tc.Kind)

	printHook := &collectingPrintHook{}

	r := rego.New(
		rego.Compiler(p.compiler),
		rego.Query(query),
		rego.Input(tc.Request),
		rego.Store(p.store),
		rego.PrintHook(printHook),
	)

	res, err := r.Eval(ctx)
	if err != nil {
		return false, "", err
	}

	prints := strings.Join(printHook.messages, "\n")

	if len(res) == 0 {
		return false, prints, fmt.Errorf("no rule %s found in policy", query)
	} else if len(res) != 1 {
		return false, prints, fmt.Errorf("expected exactly one result, got %d", len(res))
	} else if len(res[0].Expressions) != 1 {
		return false, prints, fmt.Errorf("expected exactly one expression, got %d", len(res[0].Expressions))
	}

	switch v := res[0].Expressions[0].Value.(type) {
	case bool:
		// Policy was evaluated and returned a boolean value.
		return v, prints, nil
	case map[string]any:
		// Policy returned a metadata response with allowed and ops keys.
		// The ops will be a list of json patch operations to apply to the state.
		allowed, ok := v["allowed"].(bool)
		if !ok {
			return false, "", fmt.Errorf("expected 'allowed' key to be a bool, got: %v", v["allowed"])
		}
		if !allowed {
			return false, prints, nil
		}
		ops, ok := v["ops"].([]any)
		if !ok {
			return false, "", fmt.Errorf("expected 'ops' key to be a slice, got: %v", v["ops"])
		}
		if err := p.applyOps(ctx, ops); err != nil {
			return false, "", fmt.Errorf("applying ops: %w", err)
		}
		return true, prints, nil
	default:
		return false, "", fmt.Errorf("unexpected result type, got %T", v)
	}
}

// applyOps applies the given json patch operations to the current state and updates the store.
func (p *OPAPolicy) applyOps(ctx context.Context, ops []any) error {
	// Read the whole data document from the store.
	data, err := storage.ReadOne(ctx, p.store, storage.RootPath)
	if err != nil {
		return err
	}

	currentBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		return err
	}

	patch, err := jsonpatch.DecodePatch(patchBytes)
	if err != nil {
		return err
	}

	updatedBytes, err := patch.Apply(currentBytes)
	if err != nil {
		return err
	}

	var updated map[string]any
	if err := json.Unmarshal(updatedBytes, &updated); err != nil {
		return err
	}

	// Replace the root data document.
	return storage.WriteOne(
		ctx,
		p.store,
		storage.ReplaceOp,
		storage.RootPath,
		updated,
	)
}
