package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &TfPlanResource{}
)

// NewTfPlanResource is a helper function to simplify the provider implementation.
func NewTfPlanResource() resource.Resource {
	return &TfPlanResource{}
}

// TfPlanResource is the resource implementation.
type TfPlanResource struct{}

// TfPlanResourceModel maps the resource schema data.
type TfPlanResourceModel struct {
	Path    types.String `tfsdk:"path"`
	Outputs types.Map    `tfsdk:"outputs"`
}

func (r *TfPlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName
}

func (r *TfPlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the plan outputs from an upstream Terraform stack.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "The path to the upstream Terraform stack directory.",
				Required:    true,
			},
			"outputs": schema.MapAttribute{
				Description: "The outputs from the upstream Terraform stack.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					newMapPlanModifier(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial state.
func (r *TfPlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TfPlanResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	outputs, err := r.readApplyOutputs(plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading upstream apply outputs", err.Error())
		return
	}

	plan.Outputs = outputs
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TfPlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TfPlanResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	outputs, err := r.readApplyOutputs(state.Path.ValueString())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}

		resp.Diagnostics.AddError("Error reading upstream apply outputs", err.Error())
		return
	}

	state.Outputs = outputs
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated state.
func (r *TfPlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TfPlanResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	outputs, err := r.readApplyOutputs(plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading upstream apply outputs", err.Error())
		return
	}

	plan.Outputs = outputs
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes it from the Terraform state.
func (r *TfPlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *TfPlanResource) readApplyOutputs(path string) (types.Map, error) {
	outputPath := filepath.Join(path, "outputs.json")
	tflog.Debug(context.Background(), fmt.Sprintf("Reading apply outputs from %s", outputPath))

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return types.MapNull(types.StringType), fmt.Errorf("could not read outputs file: %w", err)
	}

	var outputs map[string]OutputValue
	if err := json.Unmarshal(data, &outputs); err != nil {
		return types.MapNull(types.StringType), fmt.Errorf("could not unmarshal outputs: %w", err)
	}

	outputElements := make(map[string]attr.Value)
	for k, v := range outputs {
		switch val := v.Value.(type) {
		case string:
			outputElements[k] = types.StringValue(val)
		case float64, float32, int, int32, int64:
			outputElements[k] = types.StringValue(fmt.Sprintf("%v", val))
		case bool:
			outputElements[k] = types.StringValue(fmt.Sprintf("%t", val))
		default:
			vBytes, err := json.Marshal(v.Value)
			if err != nil {
				return types.MapNull(types.StringType), fmt.Errorf("could not marshal output value for %s: %w", k, err)
			}
			outputElements[k] = types.StringValue(string(vBytes))
		}
	}

	result, diags := types.MapValue(types.StringType, outputElements)
	if diags.HasError() {
		return types.MapNull(types.StringType), fmt.Errorf("could not create map value: %v", diags)
	}
	return result, nil
}
