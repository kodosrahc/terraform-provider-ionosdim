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
	_ resource.Resource                = &cnameRecordResource{}
	_ resource.ResourceWithConfigure   = &cnameRecordResource{}
	_ resource.ResourceWithImportState = &cnameRecordResource{}
)

func NewCNAMERecordResource() resource.Resource {
	return &cnameRecordResource{}
}

type cnameRecordResource struct {
	client *dim.Client
}

type cnameRecordResourceModel struct {
	// common identifying RR attributes
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Zone types.String `tfsdk:"zone"`
	View types.String `tfsdk:"view"` // DIM api function rr_create has "views" plural, allowing to specify multiple view at one call. We stick to 1:1 terraform:real_world mapping
	// A record specific identifying attributes
	CNAME types.String `tfsdk:"cname"`
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

type cnameRecordID struct {
	zone  string
	view  string
	name  string
	cname string
}

func newCnameRecordIDFromString(s string) (*cnameRecordID, error) {
	idParts := strings.SplitN(s, "/", 4)
	if len(idParts) != 4 {
		return nil, fmt.Errorf("ID is not in expected format")
	}
	return &cnameRecordID{
		zone:  idParts[0],
		view:  idParts[1],
		name:  idParts[2],
		cname: idParts[3],
	}, nil

}

func newCnameRecordIDFromTfModel(ctx context.Context, m cnameRecordResourceModel) (*cnameRecordID, error) {
	return &cnameRecordID{
		zone:  m.Zone.ValueString(),
		view:  m.View.ValueString(),
		name:  m.Name.ValueString(),
		cname: m.CNAME.ValueString(),
	}, nil
}

func (id cnameRecordID) String() string {
	// <zone>/<view>/<name>/<cname>
	return fmt.Sprintf("%s/%s/%s/%s", id.zone, id.view, id.name, id.cname)
}

func (id cnameRecordID) getFqdn() string {
	if name := id.name; strings.HasSuffix(name, ".") {
		return name
	} else {
		return fmt.Sprintf("%s.%s.", name, id.zone)
	}
}

// copyIDToModel set attributes in the model to values
// from which the ID was composed
func (r cnameRecordResource) restoreIDAttributesToModel(ctx context.Context, id cnameRecordID, rm *cnameRecordResourceModel) error {
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
	// required args
	rm.Name = types.StringValue(id.name)
	rm.CNAME = types.StringValue(id.cname)
	return nil
}

// readInDimResponse parse the DIM response into the Resource Model
func (r cnameRecordResource) readInDimResponse(dimResp map[string]any, rm *cnameRecordResourceModel) {
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

func (r *cnameRecordResource) diagErrorSummaryTemplate() string {
	return "Error in %s CNAME record"
}

func (r *cnameRecordResource) diagErrorDetailTemplate() string {
	return "Unexpected error from %s: %s"
}

// dimRawCall is a helper function to call DIM API and handle errors and logging
func (r *cnameRecordResource) dimRawCall(ctx context.Context, tfAction string, dfunc string, dargs []any, diags *diag.Diagnostics) (any, error) {
	tflog.Debug(ctx, fmt.Sprintf("%s/%s call", tfAction, dfunc), map[string]any{"func": dfunc, "args": dargs})
	dimResp, err := r.client.RawCall(dfunc, dargs)
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
func (r *cnameRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *cnameRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cname_record"
}

// Schema defines the schema for the resource.
func (r *cnameRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a CNAME type record in DIM.",
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
				},
				MarkdownDescription: "optional if name is a fqdn",
			},
			"view": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"cname": schema.StringAttribute{
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
			"rr": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *cnameRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Create a new resource.
	// Retrieve values from data
	var data cnameRecordResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// required args
	id, err := newCnameRecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Create"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
		return
	}

	dim_req_args := map[string]any{
		"type":  "CNAME",
		"name":  id.name,
		"cname": id.cname,
	}
	// optional args
	// see https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values#when-can-a-value-be-unknown-or-null

	// optional, computed
	if !(data.Zone.IsNull() || data.Zone.IsUnknown()) {
		dim_req_args["zone"] = id.zone
	}
	// optional
	if !(data.View.IsNull() || data.View.IsUnknown()) {
		dim_req_args["view"] = id.view
	}
	// optional
	if !(data.Comment.IsNull() || data.Comment.IsUnknown()) {
		dim_req_args["comment"] = data.Comment.ValueString()
	}
	// optional
	if !(data.TTL.IsNull() || data.TTL.IsUnknown()) {
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
	// so to reuse request args, we will remove some of them
	delete(dim_req_args, "ttl")
	delete(dim_req_args, "comment")
	delete(dim_req_args, "zone")
	dim_req_args["name"] = id.getFqdn()
	dimResp, err := r.dimRawCall(ctx, "Create",
		"rr_get_attrs",
		[]any{
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if err != nil {
		return
	}
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
func (r *cnameRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current data
	var data cnameRecordResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newCnameRecordIDFromString(data.ID.ValueString())
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
		"type":  "CNAME",
		"name":  id.getFqdn(), // rr_set_attrs has no "zone" attr, so "name" must be fqdn with trailing dot
		"cname": id.cname,
	}
	// optional args
	if id.view != "" {
		dim_req_args["view"] = id.view
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
func (r *cnameRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// only TTL and comment are updatable
	// other args result in resource replacement

	// Retrieve values from data
	var data cnameRecordResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// required args
	id, err := newCnameRecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Update"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
		return
	}
	dim_req_args := map[string]any{
		"type":  "CNAME",
		"name":  id.getFqdn(), // rr_set_attrs has no "zone" attr, so "name" must be fqdn with trailing dot
		"cname": id.cname,
	}
	// optional args
	if !data.View.IsNull() {
		dim_req_args["view"] = id.view
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
func (r *cnameRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from data
	var data cnameRecordResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newCnameRecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
			err.Error(),
		)
		return
	}

	// required args
	dim_req_args := map[string]any{
		"type":  "CNAME",
		"name":  id.name,
		"cname": id.cname,
	}
	// optional args
	if id.zone != "" {
		dim_req_args["zone"] = id.zone
	}
	if id.view != "" {
		dim_req_args["view"] = id.view
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

func (r *cnameRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
