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
	RR         types.String `tfsdk:"rr"`
}

type aRecordID struct {
	zone         string
	view         string
	name         string
	layer3domain string
	ip           string
}

func newARecordIDFromString(s string) (*aRecordID, error) {
	idParts := strings.SplitN(s, "/", 5)
	if len(idParts) != 5 {
		return nil, fmt.Errorf("ID is not in expected format")
	}
	return &aRecordID{
		zone:         idParts[0],
		view:         idParts[1],
		name:         idParts[2],
		layer3domain: idParts[3],
		ip:           idParts[4],
	}, nil
}

func newARecordIDFromTfModel(ctx context.Context, m aRecordResourceModel) (*aRecordID, error) {
	return &aRecordID{
		zone:         m.Zone.ValueString(),
		view:         m.View.ValueString(),
		name:         m.Name.ValueString(),
		layer3domain: m.Layer3domain.ValueString(),
		ip:           m.Ip.ValueString(),
	}, nil
}

func (id aRecordID) String() string {
	// <zone>/<view>/<name>/<layer3domain>/<ip>
	return fmt.Sprintf("%s/%s/%s/%s/%s", id.zone, id.view, id.name, id.layer3domain, id.ip)
}

func (id aRecordID) getFqdn() string {
	if name := id.name; strings.HasSuffix(name, ".") {
		return name
	} else {
		return fmt.Sprintf("%s.%s.", name, id.zone)
	}
}

// copyIDToModel set attributes in the model to values
// from which the ID was composed
func (r aRecordResource) restoreIDAttributesToModel(ctx context.Context, id aRecordID, rm *aRecordResourceModel) error {
	// optional args
	if id.zone != "" {
		rm.Zone = types.StringValue(id.zone)
	} else {
		rm.Zone = types.StringNull()
	}
	if id.view != "" {
		rm.View = types.StringValue(id.view)
	} else {
		rm.View = types.StringNull()
	}
	if id.layer3domain != "" {
		rm.Layer3domain = types.StringValue(id.layer3domain)
	} else {
		rm.Layer3domain = types.StringNull()
	}
	// required args
	rm.Name = types.StringValue(id.name)
	rm.Ip = types.StringValue(id.ip)
	return nil
}

func (r aRecordResource) readInDimResponse(dimResp map[string]any, rm *aRecordResourceModel) {
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
	if v, ok := dimResp["rr"]; ok {
		rm.RR = types.StringValue(v.(string))
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
	dimResp, err := r.client.RawCallWithContext(ctx, dfunc, dargs)
	if err != nil {
		if diags != nil {
			diags.AddError(
				fmt.Sprintf(r.diagErrorSummaryTemplate(), tfAction),
				fmt.Sprintf(r.diagErrorDetailTemplate(), dfunc, err.Error()),
			)
		}
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
		MarkdownDescription: "Creates a A type record in DIM.",
		Attributes: map[string]schema.Attribute{
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
			"zone": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "optional if name is a fqdn",
			},
			"view": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"layer3domain": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "value is optional when specifying a RR if there is only one RR with that name, type and value",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"modified": schema.StringAttribute{
				Computed: true,
			},
			"modified_by": schema.StringAttribute{
				Computed: true,
			},
			"rr": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *aRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Create a new resource.
	// Retrieve values from data
	var data aRecordResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newARecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Create"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
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
			id.ip,
			map[string]any{
				"host":         true,
				"layer3domain": id.layer3domain,
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
			fmt.Sprintf("IP address %s is not allocated (not marked as Static)", id.ip),
		)
		return
	}
	//now we know that the IP is allocated

	// required args
	dim_req_args := map[string]any{
		"type": "A",
		"name": id.name,
		"ip":   id.ip,
	}
	// optional args
	// see https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values#when-can-a-value-be-unknown-or-null

	// optional
	if !(data.Layer3domain.IsNull() || data.Layer3domain.IsUnknown()) {
		dim_req_args["layer3domain"] = id.layer3domain
	}
	// optional, computed
	if !(data.Zone.IsNull() || data.Zone.IsUnknown()) {
		dim_req_args["zone"] = id.zone
	}
	// optional
	if !(data.View.IsNull() || data.View.IsUnknown()) {
		dim_req_args["view"] = id.view
	}
	// optional
	if !data.Comment.IsNull() {
		dim_req_args["comment"] = data.Comment.ValueString()
	}
	// optional
	if !data.TTL.IsNull() {
		dim_req_args["ttl"] = data.TTL.ValueInt64()
	}

	_, err = r.dimRawCall(ctx, "Create",
		"rr_create",
		[]any{
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if err != nil {
		return
	}
	tflog.Info(ctx, "RR has been created", dim_req_args)

	//rr_get_attrs
	// rr_get_attrs accepts the subset of the args of rr_create
	// so to reuse them, we will remove some of them
	delete(dim_req_args, "ttl")
	delete(dim_req_args, "comment")
	delete(dim_req_args, "zone")
	dim_req_args["name"] = id.getFqdn() // rr_get_attrs has no "zone" arg, so "name" arg must be fqdn
	dimResp, err = r.dimRawCall(ctx, "Create",
		"rr_get_attrs",
		[]any{
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if err != nil {
		return
	}
	// not all attributes of a RR are returned by rr_get_attrs,
	// see the example response in ../../docs/.dim/rr_get_attrs.md
	r.readInDimResponse(dimResp.(map[string]any), &data)

	// now when we know the all values, set the ID
	data.ID = types.StringValue(id.String())

	// Set state to fully populated data
	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *aRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current data
	var data aRecordResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newARecordIDFromString(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("ID parsed %+v", id))
	r.restoreIDAttributesToModel(ctx, *id, &data)

	// required args
	dim_req_args := map[string]any{
		"type": "A",
		"name": id.getFqdn(), // rr_set_attrs has no "zone" attr, so "name" must be fqdn with trailing dot
		"ip":   id.ip,
	}
	// optional args
	if id.view != "" {
		dim_req_args["view"] = id.view
	}
	if id.layer3domain != "" {
		dim_req_args["layer3domain"] = id.layer3domain
	}

	tflog.Info(ctx, "Will read RR", dim_req_args)
	dimResp, err := r.dimRawCall(ctx, "Read",
		"rr_get_attrs",
		[]any{
			dim_req_args,
		},
		nil,
	)
	if err != nil {
		if _, ok := err.(dim.Error); ok {
			if err.(dim.Error).Code == 1 {
				tflog.Debug(ctx, fmt.Sprintf("record not found (has been removed?) %+v", id))
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			fmt.Sprintf(r.diagErrorDetailTemplate(), "rr_get_attrs", err.Error()),
		)
		return
	}
	r.readInDimResponse(dimResp.(map[string]any), &data)
	tflog.Info(ctx, "RR has been read", dim_req_args)

	// Set refreshed state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *aRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// only TTL and comment are updatable
	// other args result in resource replacement

	// Retrieve values from data
	var data aRecordResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// required args
	id, err := newARecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Update"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
		return
	}
	dim_req_args := map[string]any{
		"type": "A",
		"name": id.getFqdn(), // rr_set_attrs has no "zone" attr, so "name" must be fqdn with trailing dot
		"ip":   id.ip,
	}
	// optional args
	if !data.View.IsNull() {
		dim_req_args["view"] = id.view
	}
	if !data.Layer3domain.IsNull() {
		dim_req_args["layer3domain"] = id.layer3domain
	}
	//updatable args
	if !data.TTL.IsNull() {
		dim_req_args["ttl"] = data.TTL.ValueInt64()
	}
	if !data.Comment.IsNull() {
		dim_req_args["comment"] = data.Comment.ValueString()
	}

	tflog.Info(ctx, "Will update RR", dim_req_args)
	dimResp, _ := r.dimRawCall(ctx, "Update",
		"rr_set_attrs",
		[]any{
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInDimResponse(dimResp.(map[string]any), &data)
	tflog.Info(ctx, "RR has been updated", dim_req_args)

	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *aRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from data
	var data aRecordResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newARecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
			err.Error(),
		)
		return
	}

	// required args
	dim_req_args := map[string]any{
		"type": "A",
		"name": id.name,
		"ip":   id.ip,
	}
	// optional args
	if id.zone != "" {
		dim_req_args["zone"] = id.zone
	}
	if id.view != "" {
		dim_req_args["view"] = id.view
	}
	if id.layer3domain != "" {
		dim_req_args["layer3domain"] = id.layer3domain
	}
	// delete specific args
	dim_req_args["references"] = "warn"

	//	_, err := i.client.RawCall("rr_delete", []any{dim_req_args})
	r.dimRawCall(ctx, "Delete",
		"rr_delete",
		[]any{
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *aRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
