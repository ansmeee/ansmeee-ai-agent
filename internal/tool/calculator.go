package tool

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
)

// Calculator evaluates simple math expressions.
type Calculator struct{}

// Name returns the tool name.
func (c *Calculator) Name() string { return "calculator" }

// Description returns the tool description.
func (c *Calculator) Description() string {
	return "Evaluate a mathematical expression. Supports +, -, *, /, %, ^ (power), sqrt, abs. Example: \"2 + 3 * 4\""
}

// Parameters returns the JSON Schema for the tool's input.
func (c *Calculator) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "Mathematical expression to evaluate, e.g. \"2 + 3 * 4\"",
			},
		},
		"required": []string{"expression"},
	}
}

// Call evaluates the given math expression.
func (c *Calculator) Call(ctx context.Context, input string) (string, error) {
	expr, err := parser.ParseExpr(input)
	if err != nil {
		return "", fmt.Errorf("invalid expression %q: %w", input, err)
	}
	result, err := evalExpr(expr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%g", result), nil
}

func evalExpr(expr ast.Expr) (float64, error) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return evalBinary(e)
	case *ast.UnaryExpr:
		return evalUnary(e)
	case *ast.BasicLit:
		if e.Kind == token.INT || e.Kind == token.FLOAT {
			return strconv.ParseFloat(e.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal: %s", e.Value)
	case *ast.ParenExpr:
		return evalExpr(e.X)
	case *ast.CallExpr:
		return evalCall(e)
	case *ast.Ident:
		if e.Name == "pi" {
			return math.Pi, nil
		}
		if e.Name == "e" {
			return math.E, nil
		}
		return 0, fmt.Errorf("unknown identifier: %s", e.Name)
	default:
		return 0, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func evalBinary(expr *ast.BinaryExpr) (float64, error) {
	left, err := evalExpr(expr.X)
	if err != nil {
		return 0, err
	}
	right, err := evalExpr(expr.Y)
	if err != nil {
		return 0, err
	}
	switch expr.Op {
	case token.ADD:
		return left + right, nil
	case token.SUB:
		return left - right, nil
	case token.MUL:
		return left * right, nil
	case token.QUO:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case token.REM:
		if right == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}
		return math.Mod(left, right), nil
	case token.XOR:
		return math.Pow(left, right), nil
	default:
		return 0, fmt.Errorf("unsupported operator: %s", expr.Op)
	}
}

func evalUnary(expr *ast.UnaryExpr) (float64, error) {
	x, err := evalExpr(expr.X)
	if err != nil {
		return 0, err
	}
	switch expr.Op {
	case token.SUB:
		return -x, nil
	case token.ADD:
		return x, nil
	default:
		return 0, fmt.Errorf("unsupported unary operator: %s", expr.Op)
	}
}

func evalCall(expr *ast.CallExpr) (float64, error) {
	fn, ok := expr.Fun.(*ast.Ident)
	if !ok {
		return 0, fmt.Errorf("unsupported function call")
	}
	if len(expr.Args) != 1 {
		return 0, fmt.Errorf("%s requires exactly 1 argument", fn.Name)
	}
	arg, err := evalExpr(expr.Args[0])
	if err != nil {
		return 0, err
	}
	switch fn.Name {
	case "sqrt":
		if arg < 0 {
			return 0, fmt.Errorf("sqrt of negative number")
		}
		return math.Sqrt(arg), nil
	case "abs":
		return math.Abs(arg), nil
	default:
		return 0, fmt.Errorf("unknown function: %s", fn.Name)
	}
}
