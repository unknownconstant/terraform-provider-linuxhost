package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringOrNull(value interface{}) types.String {
	if value == nil {
		return types.StringNull()
	}

	switch v := value.(type) {
	case string:
		return types.StringValue(v)
	case *string:
		if v == nil {
			return types.StringNull()
		}
		return types.StringValue(*v)
	default:
		return types.StringNull()
	}
}

func numberOrNull(value interface{}) types.Number {
	fmt.Println("Checking number!")
	fmt.Println(value)
	if value == nil {
		return types.NumberNull()
	}

	switch v := value.(type) {
	case int:
		fmt.Println(v)
		return types.NumberValue(big.NewFloat(float64(v))) // Convert int to float64, then to *big.Float
	case *int:
		fmt.Println(*v)
		return types.NumberValue(big.NewFloat(float64(*v)))
	case float64:
		fmt.Println(v)
		return types.NumberValue(big.NewFloat(v)) // Directly use float64
	case *float64:
		fmt.Println(*v)
		return types.NumberValue(big.NewFloat(*v)) // Handle pointer to float64

	default:
		fmt.Println("DEFAULT!! Unkown type!!")
		// Return null if the type is unsupported
		return types.NumberNull()
	}
}

func int64OrNull(value interface{}) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	switch v := value.(type) {
	case int64:
		return types.Int64Value(v)
	case *int64:
		if v == nil {
			return types.Int64Null()
		}
		return types.Int64Value(*v)
	default:
		return types.Int64Null()
	}
}
func int32OrNull(value interface{}) types.Int32 {
	if value == nil {
		return types.Int32Null()
	}
	switch v := value.(type) {
	case uint32:
		return types.Int32Value(int32(v))
	case *uint32:
		if v == nil {
			return types.Int32Null()
		}
		return types.Int32Value(int32(*v))
	case int32:
		return types.Int32Value(v)
	case *int32:
		if v == nil {
			return types.Int32Null()
		}
		return types.Int32Value(*v)
	default:
		return types.Int32Null()
	}
}

type ReadableResource[M any] struct {
	Equal        func(A *M, B *M) bool
	New          func(current *M, target *M) M
	Ctx          context.Context
	State        *tfsdk.State
	Diagnostics  *diag.Diagnostics
	currentState []M
}

func (r ReadableResource[M]) InState(target M, expect string) {
	for _, current := range r.currentState {
		if !r.Equal(&current, &target) {
			continue
		}
		if expect == "absent" {
			r.Diagnostics.AddError("Failed to delete", "The delete operation did not report any errors but the resource remains present in the reported state.")
			return
		}
		new := r.New(&current, &target)
		r.Diagnostics.Append(r.State.Set(r.Ctx, new)...)
		return
	}
	if expect == "present" {
		r.Diagnostics.AddError("Didn't find resource", "")
	} else if expect == "any" || expect == "absent" {
		r.State.RemoveResource(r.Ctx)
	} else {
		r.Diagnostics.AddError("Invalid expectation", "This is an error with the provider 'linuxhost'")
	}
}
