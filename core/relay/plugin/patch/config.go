// Package patch provides configuration types and structures for the patch plugin.
package patch

import "github.com/bytedance/sonic/ast"

// Config holds the configuration for the patch plugin
type Config struct {
	// Force Enabled

	// UserPatches are user-defined custom patches
	UserPatches []PatchRule `json:"user_patches,omitempty"`
}

// PatchRule defines a complete patch rule with conditions and operations
type PatchRule struct {
	// Name is a descriptive name for this patch rule
	Name string `json:"name"`
	// Description explains what this patch does
	Description string `json:"description,omitempty"`
	// Conditions determine when this patch should be applied
	Conditions []PatchCondition `json:"conditions,omitempty"`
	// Operations define what modifications to make
	Operations []PatchOperation `json:"operations"`
}

// PatchCondition defines when a patch should be applied
type PatchCondition struct {
	// Key is the field to check (supports dot notation for nested fields)
	// Special keys: "model", "original_model"
	Key string `json:"key"`
	// Operator defines how to compare the value
	Operator ConditionOperator `json:"operator"`
	// Value is the value to compare against
	Value string `json:"value"`
	// Values is an array of values for 'in' and 'not_in' operators
	Values []string `json:"values,omitempty"`
}

type PatchFunction func(root *ast.Node) (bool, error)

// PatchOperation defines a modification to make to the JSON
type PatchOperation struct {
	// Op is the operation type
	Op OperationType `json:"op"`
	// Key is the field to modify (supports dot notation for nested fields)
	Key string `json:"key"`
	// Value is the new value to set (not used for delete operations)
	Value any `json:"value,omitempty"`
	// Function is the inline function code for OpFunction operations
	function PatchFunction `json:"-"`
}

// ConditionOperator defines how to evaluate a condition
type ConditionOperator string

const (
	OperatorEquals      ConditionOperator = "equals"
	OperatorNotEquals   ConditionOperator = "not_equals"
	OperatorContains    ConditionOperator = "contains"
	OperatorNotContains ConditionOperator = "not_contains"
	OperatorRegex       ConditionOperator = "regex"
	OperatorExists      ConditionOperator = "exists"
	OperatorNotExists   ConditionOperator = "not_exists"
	OperatorHasPrefix   ConditionOperator = "has_prefix"
	OperatorHasSuffix   ConditionOperator = "has_suffix"
	OperatorGreaterThan ConditionOperator = "greater_than"
	OperatorLessThan    ConditionOperator = "less_than"
	OperatorGreaterEq   ConditionOperator = "greater_eq"
	OperatorLessEq      ConditionOperator = "less_eq"
	OperatorIn          ConditionOperator = "in"
	OperatorNotIn       ConditionOperator = "not_in"
)

// OperationType defines the type of operation to perform
type OperationType string

const (
	// OpSet sets a field to a specific value
	OpSet OperationType = "set"
	// OpDelete removes a field
	OpDelete OperationType = "delete"
	// OpAdd adds a field only if it doesn't exist
	OpAdd OperationType = "add"
	// OpLimit limits a numeric field to a maximum value
	OpLimit OperationType = "limit"
	// OpIncrement increments a numeric field by a value
	OpIncrement OperationType = "increment"
	// OpDecrement decrements a numeric field by a value
	OpDecrement OperationType = "decrement"
	// OpMultiply multiplies a numeric field by a value
	OpMultiply OperationType = "multiply"
	// OpDivide divides a numeric field by a value
	OpDivide OperationType = "divide"
	// OpAppend appends value to an array field
	OpAppend OperationType = "append"
	// OpPrepend prepends value to an array field
	OpPrepend OperationType = "prepend"
	// OpFunction executes an inline function on the field
	OpFunction OperationType = "function"
)

// DefaultPredefinedPatches are built-in patches that are always available
var DefaultPredefinedPatches = []PatchRule{
	{
		Name:        "deepseek_max_tokens_limit",
		Description: "Limit max_tokens to 16000 for DeepSeek models",
		Conditions: []PatchCondition{
			{
				Key:      "model",
				Operator: OperatorContains,
				Value:    "deepseek",
			},
		},
		Operations: []PatchOperation{
			{
				Op:    OpLimit,
				Key:   "max_tokens",
				Value: 16000,
			},
		},
	},
	{
		Name:        "gpt5_max_tokens_to_max_completion_tokens",
		Description: "Convert max_tokens to max_completion_tokens for GPT-5 models",
		Conditions: []PatchCondition{
			{
				Key:      "model",
				Operator: OperatorContains,
				Value:    "gpt-5",
			},
			{
				Key:      "max_tokens",
				Operator: OperatorExists,
				Value:    "",
			},
		},
		Operations: []PatchOperation{
			{
				Op:    OpSet,
				Key:   "max_completion_tokens",
				Value: "{{max_tokens}}", // Special placeholder that will be replaced with actual max_tokens value
			},
			{
				Op:  OpDelete,
				Key: "max_tokens",
			},
		},
	},
	{
		Name:        "o1_max_tokens_to_max_completion_tokens",
		Description: "Convert max_tokens to max_completion_tokens for o1 models",
		Conditions: []PatchCondition{
			{
				Key:      "model",
				Operator: OperatorRegex,
				Value:    "^o1(-preview|-mini)?$",
			},
			{
				Key:      "max_tokens",
				Operator: OperatorExists,
				Value:    "",
			},
		},
		Operations: []PatchOperation{
			{
				Op:    OpSet,
				Key:   "max_completion_tokens",
				Value: "{{max_tokens}}",
			},
			{
				Op:  OpDelete,
				Key: "max_tokens",
			},
		},
	},
	{
		Name:        "claude_max_tokens_limit",
		Description: "Limit max_tokens to reasonable values for Claude models",
		Conditions: []PatchCondition{
			{
				Key:      "model",
				Operator: OperatorContains,
				Value:    "claude",
			},
		},
		Operations: []PatchOperation{
			{
				Op:    OpLimit,
				Key:   "max_tokens",
				Value: 8192,
			},
		},
	},
	{
		Name:        "remove_unsupported_stream_options",
		Description: "Remove stream_options for models that don't support it",
		Conditions: []PatchCondition{
			{
				Key:      "model",
				Operator: OperatorRegex,
				Value:    "(gpt-3\\.5|gpt-4-(?!turbo))",
			},
			{
				Key:      "stream_options",
				Operator: OperatorExists,
				Value:    "",
			},
		},
		Operations: []PatchOperation{
			{
				Op:  OpDelete,
				Key: "stream_options",
			},
		},
	},
}
