package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-ionosdim/pkg/dim"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ipResource{}
	_ resource.ResourceWithConfigure   = &ipResource{}
	_ resource.ResourceWithImportState = &ipResource{}
)

// NewCoffeesDataSource is a helper function to simplify the provider implementation.
func NewIpResource() resource.Resource {
	return &ipResource{}
}

// coffeesDataSource is the data source implementation.
type ipResource struct {
	client *dim.Client
}

type ipResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Ip           types.String `tfsdk:"ip"`
	Cidr         types.String `tfsdk:"cidr"`
	Layer3domain types.String `tfsdk:"layer3domain"`

	Created     types.String `tfsdk:"created"` // no created_by field!
	Gateway     types.String `tfsdk:"gateway"`
	Mask        types.String `tfsdk:"mask"`
	Modified    types.String `tfsdk:"modified"`
	ModifiedBy  types.String `tfsdk:"modified_by"`
	Pool        types.String `tfsdk:"pool"`
	ReverseZone types.String `tfsdk:"reverse_zone"`
	Status      types.String `tfsdk:"status"`
	Subnet      types.String `tfsdk:"subnet"`
}

func (rm ipResourceModel) composeID() string {
	//types.StringValue(_layer3domain + "/" + _ip)
	return rm.Layer3domain.ValueString() + "/" + rm.Ip.ValueString()
}

func (rm *ipResourceModel) readInDimResponse(dimResp map[string]any) {

	// not Computed, make no sense to read from dimResp
	// if v, ok := dimResp["layer3domain"]; ok {
	// 	rm.Layer3domain = types.StringValue(v.(string))
	// }
	if v, ok := dimResp["ip"]; ok {
		rm.Ip = types.StringValue(v.(string))
	}

	if v, ok := dimResp["created"]; ok {
		rm.Created = types.StringValue(v.(string))
	}
	if v, ok := dimResp["modified"]; ok {
		rm.Modified = types.StringValue(v.(string))
	}
	if v, ok := dimResp["modified_by"]; ok {
		rm.ModifiedBy = types.StringValue(v.(string))
	}

	if v, ok := dimResp["gateway"]; ok {
		rm.Gateway = types.StringValue(v.(string))
	}
	if v, ok := dimResp["mask"]; ok {
		rm.Mask = types.StringValue(v.(string))
	}
	if v, ok := dimResp["pool"]; ok {
		rm.Pool = types.StringValue(v.(string))
	}
	if v, ok := dimResp["reverse_zone"]; ok {
		rm.ReverseZone = types.StringValue(v.(string))
	}
	if v, ok := dimResp["subnet"]; ok {
		rm.Subnet = types.StringValue(v.(string))
	}
	if v, ok := dimResp["status"]; ok {
		rm.Status = types.StringValue(v.(string))
	}

}

func (r *ipResource) diagErrorSummaryTemplate() string {
	return "Error in %s ip"
}

func (r *ipResource) diagErrorDetailTemplate() string {
	return "Unexpected error from %s: %s"
}

func (r *ipResource) diagWarningSummaryTemplate() string {
	return "Warning in %s ip"
}

func (r *ipResource) dimRawCall(ctx context.Context, tfAction string, dfunc string, dargs []any, diags *diag.Diagnostics) (any, error) {
	tflog.Debug(ctx, fmt.Sprintf("%s/%s call", tfAction, dfunc), map[string]any{"func": dfunc, "args": dargs})
	dimResp, err := r.client.RawCall(dfunc, dargs)
	if err != nil {
		diags.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), tfAction),
			fmt.Sprintf(r.diagErrorDetailTemplate(), dfunc, err.Error()),
		)
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("%s/%s response", tfAction, dfunc), map[string]any{"dimResponse": dimResp})
	return dimResp, nil
}

// Configure adds the provider configured client to the resource.
func (r *ipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dim.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *dim.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *ipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip"
}

// Schema defines the schema for the resource.
func (r *ipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"layer3domain": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"created": schema.StringAttribute{
				Computed: true,
			},
			"gateway": schema.StringAttribute{
				Computed: true,
			},
			"mask": schema.StringAttribute{
				Computed: true,
			},
			"modified": schema.StringAttribute{
				Computed: true,
			},
			"modified_by": schema.StringAttribute{
				Computed: true,
			},
			"pool": schema.StringAttribute{
				//Optional: true,
				Computed: true,
			},
			"reverse_zone": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"subnet": schema.StringAttribute{
				//Optional: true,
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	//
	// Create a new resource.
	// Retrieve values from plan
	var plan ipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dim_req_args := map[string]any{}

	if !plan.Layer3domain.IsNull() {
		dim_req_args["layer3domain"] = plan.Layer3domain.ValueString()
		ctx = tflog.SetField(ctx, "layer3domain", plan.Layer3domain.ValueString())
	}

	var dimResp any

	if plan.Ip.IsUnknown() {
		ctx = tflog.SetField(ctx, "cidr", plan.Cidr.ValueString())
		dimResp, _ = r.dimRawCall(ctx, "Create", "ipblock_get_ip", []any{plan.Cidr.ValueString(), dim_req_args}, &resp.Diagnostics)
	} else {
		ctx = tflog.SetField(ctx, "ip", plan.Ip.ValueString())
		dimResp, _ = r.dimRawCall(ctx, "Create", "ip_mark", []any{plan.Ip.ValueString(), dim_req_args}, &resp.Diagnostics)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IP has been made static", dim_req_args)

	plan.readInDimResponse(dimResp.(map[string]any))
	// now when we know the all values, set the ID
	plan.ID = types.StringValue(plan.composeID())

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *ipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state ipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	idParts := strings.SplitN(state.ID.ValueString(), "/", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			"ID is not in expected format",
		)
		return
	}

	dim_req_args := map[string]any{
		"host":         true,
		"layer3domain": idParts[0],
	}

	ctx = tflog.SetField(ctx, "cidr", idParts[1])
	tflog.Info(ctx, "Will read IP", dim_req_args)
	dimResp, _ := r.dimRawCall(
		ctx, "Read", "ipblock_get_attrs",
		[]any{
			idParts[1], // cidr
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	state.readInDimResponse(dimResp.(map[string]any))

	if state.ID.ValueString() != state.composeID() {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			fmt.Sprintf("ID has changed, old=%s, new=%s", state.ID.ValueString(), state.composeID()),
		)
		return
	}
	if state.Status.ValueString() != "Static" {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			"IP is not in Static state",
		)
		return
	}
	tflog.Info(ctx, "IP has been read", dim_req_args)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	///

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Retrieve values from state
	var state ipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idParts := strings.SplitN(state.ID.ValueString(), "/", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
			"ID is not in expected format",
		)
		return
	}

	dim_req_args := map[string]any{
		"layer3domain": idParts[0],
	}

	dimResp, _ := r.dimRawCall(
		ctx, "Delete", "ip_free",
		[]any{
			idParts[1], // cidr
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	if _res := int(dimResp.(float64)); _res != 1 {
		if _res == -1 {
			resp.Diagnostics.AddError(
				fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
				"Freeing reserved IP is not supported by the provider",
			)
			return
		}

		if _res < -1 || _res > 1 {
			resp.Diagnostics.AddError(
				fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
				fmt.Sprintf("Unexpected result from ip_free: %d", _res),
			)
			return
		}

		if _res == 0 {
			resp.Diagnostics.AddWarning(
				fmt.Sprintf(r.diagWarningSummaryTemplate(), "Delete"),
				"IP was already free",
			)
		}
	}

}

func (r *ipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
