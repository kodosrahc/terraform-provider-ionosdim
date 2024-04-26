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
	_ resource.Resource                = &aRecordResource{}
	_ resource.ResourceWithConfigure   = &aRecordResource{}
	_ resource.ResourceWithImportState = &aRecordResource{}
)

func NewARecordResource() resource.Resource {
	return &aRecordResource{}
}

type aRecordResource struct {
	client *dim.Client
}

type aRecordResourceModel struct {
	// common identifying RR attributes
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Layer3domain types.String `tfsdk:"layer3domain"`
	Zone         types.String `tfsdk:"zone"`
	View         types.String `tfsdk:"view"` // DIM api function rr_create has "views" plural, allowing to specify multiple view at one call. We stick to 1:1 terraform:real_world mapping
	// A record specific identifying attributes
	Ip types.String `tfsdk:"ip"`
	// common non-identifying changeable attributes
	Comment types.String `tfsdk:"comment"`
	TTL     types.Int64  `tfsdk:"ttl"`
	// common computed attributes
	Created    types.String `tfsdk:"created"`
	CreatedBy  types.String `tfsdk:"created_by"`
	Modified   types.String `tfsdk:"modified"`
	ModifiedBy types.String `tfsdk:"modified_by"`
}

func (rm aRecordResourceModel) composeID() string {
	return rm.Layer3domain.ValueString() + "/" + rm.Zone.ValueString() + "/" + rm.View.ValueString() + "/" + rm.Name.ValueString() + "/" + rm.Ip.ValueString()
}

func (rm *aRecordResourceModel) readInDimResponse(dimResp map[string]any) {

	// should it be replaced with json.Unmarshal?

	if v, ok := dimResp["created"]; ok {
		rm.Created = types.StringValue(v.(string))
	}
	if v, ok := dimResp["created_by"]; ok {
		rm.CreatedBy = types.StringValue(v.(string))
	}
	if v, ok := dimResp["modified"]; ok {
		rm.Modified = types.StringValue(v.(string))
	}
	if v, ok := dimResp["modified_by"]; ok {
		rm.ModifiedBy = types.StringValue(v.(string))
	}

	if v, ok := dimResp["zone"]; ok {
		rm.Zone = types.StringValue(v.(string))
	}

	// read other OPTIONAl attributes
	if v, ok := dimResp["TTL"]; ok {
		rm.TTL = types.Int64Value(v.(int64))
	}
	if v, ok := dimResp["comment"]; ok {
		rm.Comment = types.StringValue(v.(string))
	}

	// not confirmed, whether this attributes are returned
	if v, ok := dimResp["view"]; ok {
		rm.View = types.StringValue(v.(string))
	}

}

func (r *aRecordResource) diagErrorSummaryTemplate() string {
	return "Error in %s A record"
}

func (r *aRecordResource) diagErrorDetailTemplate() string {
	return "Unexpected error from %s: %s"
}

// dimRawCall is a helper function to call DIM API and handle errors and logging
func (r *aRecordResource) dimRawCall(ctx context.Context, tfAction string, dfunc string, dargs []any, diags *diag.Diagnostics) (any, error) {
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
func (r *aRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *aRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_a_record"
}

// Schema defines the schema for the resource.
func (r *aRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// "<layer3domain>/<zone>/<view>/<name>/<ip>"
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "the fqdn of the RR or the relative name if zone was specified",
			},
			"layer3domain": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "value is optional when specifying a RR if there is only one RR with that name, type and value",
			},
			"zone": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "optional if name is a fqdn",
			},
			"view": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"ip": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"comment": schema.StringAttribute{
				Optional: true,
			},
			"ttl": schema.Int64Attribute{
				Optional: true,
			},

			"created": schema.StringAttribute{
				Computed: true,
			},
			"created_by": schema.StringAttribute{
				Computed: true,
			},
			"modified": schema.StringAttribute{
				Computed: true,
			},
			"modified_by": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *aRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	//
	// Create a new resource.
	// Retrieve values from plan
	var plan aRecordResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// rr_create will allocate (set to Static) the specified IP address,
	// if the latter is not yet allocated.
	// As the result there will be an IP address not tracked by Terraform.
	// To avoid this, we will check if the IP address is already allocated,
	// if it's not, we will reject the plan.

	// check if the IP is already allocated
	dimResp, _ := r.dimRawCall(
		ctx, "Create", "ipblock_get_attrs",
		[]any{
			plan.Ip.ValueString(),
			map[string]any{
				"host":         true,
				"layer3domain": plan.Layer3domain.ValueString(),
			},
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	if dimResp.(map[string]any)["status"] != "Static" {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Create"),
			fmt.Sprintf("IP address %s is not allocated (not marked as Static)", plan.Ip.ValueString()),
		)
		return
	}
	//now we know that the IP is allocated

	// required args
	dim_req_args := map[string]any{
		"type": "A",
		"name": plan.Name.ValueString(),
		"ip":   plan.Ip.ValueString(),
	}
	// optional args
	// see https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values#when-can-a-value-be-unknown-or-null

	// optional
	if !plan.Layer3domain.IsNull() {
		dim_req_args["layer3domain"] = plan.Layer3domain.ValueString()
	}
	// optional, computed
	if !plan.Zone.IsUnknown() {
		dim_req_args["zone"] = plan.Zone.ValueString()
	}
	// optional
	if !plan.View.IsNull() {
		dim_req_args["view"] = plan.View.ValueString()
	}
	// optional
	if !plan.Comment.IsNull() {
		dim_req_args["comment"] = plan.Comment.ValueString()
	}
	// optional
	if !plan.TTL.IsNull() {
		dim_req_args["ttl"] = plan.TTL.ValueInt64()
	}

	_, err := r.dimRawCall(ctx, "Create", "rr_create", []any{dim_req_args}, &resp.Diagnostics)
	if err != nil {
		return
	}
	tflog.Info(ctx, "RR has been created", dim_req_args)

	//rr_get_attrs
	// rr_get_attrs accepts the subset of the args of rr_create
	// so to reuse them, we will remove some of them
	delete(dim_req_args, "ttl")
	delete(dim_req_args, "comment")

	dimResp, err = r.dimRawCall(ctx, "Create", "rr_get_attrs", []any{dim_req_args}, &resp.Diagnostics)
	if err != nil {
		return
	}
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
func (r *aRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state aRecordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	//"<layer3domain>/<zone>/<view>/<name>/<ip>
	idParts := strings.SplitN(state.ID.ValueString(), "/", 5)
	if len(idParts) != 5 {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			"ID is not in expected format",
		)
		return
	}

	// required args
	dim_req_args := map[string]any{
		"type":         "A",
		"layer3domain": idParts[0], // layer3domain

		"name": idParts[3], // name
		"ip":   idParts[4], // ip
	}
	// optional args
	if idParts[2] != "" {
		dim_req_args["view"] = idParts[2]
	}

	tflog.Info(ctx, "Will read RR", dim_req_args)

	dimResp, err := r.dimRawCall(ctx, "Read", "rr_get_attrs",
		[]any{dim_req_args},
		&resp.Diagnostics,
	)
	if err != nil {
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

	tflog.Info(ctx, "RR has been read", dim_req_args)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	///

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *aRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// only TTL and comment are updatable
	// other args result in resource replacement

	// Retrieve values from plan
	var plan aRecordResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name, ip := plan.Name.ValueString(), plan.Ip.ValueString()
	layer3domain, zone, view := plan.Layer3domain.ValueString(), plan.Zone.ValueString(), plan.View.ValueString()

	// required args
	dim_req_args := map[string]any{
		"type": "A",
		"name": name,
		"ip":   ip,
	}
	// optional args
	if layer3domain != "" {
		dim_req_args["layer3domain"] = layer3domain
	}
	if zone != "" {
		dim_req_args["zone"] = zone
	}
	if view != "" {
		dim_req_args["view"] = view
	}

	//updatable args
	if !plan.TTL.IsNull() {
		dim_req_args["ttl"] = plan.TTL.ValueInt64()
	}
	if !plan.Comment.IsNull() {
		dim_req_args["comment"] = plan.Comment.ValueString()
	}

	tflog.Info(ctx, "Will update RR", dim_req_args)
	dimResp, _ := r.dimRawCall(ctx, "Update", "rr_set_attrs", []any{dim_req_args}, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.readInDimResponse(dimResp.(map[string]any))

	tflog.Info(ctx, "RR has been updated", dim_req_args)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *aRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Retrieve values from state
	var state aRecordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idParts := strings.SplitN(state.ID.ValueString(), "/", 5)
	if len(idParts) != 5 {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
			"ID is not in expected format",
		)
		return
	}

	// required args
	dim_req_args := map[string]any{
		"type":         "A",
		"layer3domain": idParts[0], // layer3domain

		"name": idParts[3], // name
		"ip":   idParts[4], // ip
	}
	// optional args
	if idParts[1] != "" {
		dim_req_args["zone"] = idParts[1]
	}
	if idParts[2] != "" {
		dim_req_args["view"] = idParts[2]
	}
	// delete specific args
	dim_req_args["references"] = "warn"

	//	_, err := i.client.RawCall("rr_delete", []any{dim_req_args})
	r.dimRawCall(ctx, "Delete", "rr_delete", []any{dim_req_args}, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *aRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
