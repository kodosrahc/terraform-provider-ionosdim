package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"terraform-provider-ionosdim/pkg/dim"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &txtRecordResource{}
	_ resource.ResourceWithConfigure   = &txtRecordResource{}
	_ resource.ResourceWithImportState = &txtRecordResource{}
)

func NewTXTRecordResource() resource.Resource {
	return &txtRecordResource{}
}

type txtRecordResource struct {
	client *dim.Client
}

type txtRecordResourceModel struct {
	// common identifying RR attributes
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Zone types.String `tfsdk:"zone"`
	View types.String `tfsdk:"view"` // DIM api function rr_create has "views" plural, allowing to specify multiple view at one call. We stick to 1:1 terraform:real_world mapping
	// A record specific identifying attributes
	Strings types.List `tfsdk:"strings"`
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

type txtRecordID struct {
	zone    string
	view    string
	name    string
	strings []string
}

func newTxtRecordIDFromString(s string) (*txtRecordID, error) {
	idParts := strings.SplitN(s, "/", 4)
	if len(idParts) != 4 {
		return nil, fmt.Errorf("ID is not in expected format")
	}
	// parse "strings" part of the ID
	id_ss := strings.Split(idParts[3], ",")
	strs := make([]string, len(id_ss))
	for i, s := range id_ss {
		var err error
		strs[i], err = url.QueryUnescape(s)
		if err != nil {
			return nil, fmt.Errorf("ID is not in expected format, strings part is not URL encoded")
		}
	}
	return &txtRecordID{
		zone:    idParts[0],
		view:    idParts[1],
		name:    idParts[2],
		strings: strs,
	}, nil

}

func newTxtRecordIDFromTfModel(ctx context.Context, m txtRecordResourceModel) (*txtRecordID, error) {
	newID := txtRecordID{
		zone: m.Zone.ValueString(),
		view: m.View.ValueString(),
		name: m.Name.ValueString(),
	}
	//elements := make([]string, 0, len(m.Strings.Elements()))
	var elements []string
	diags := m.Strings.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, fmt.Errorf("%s (%s)", diags.Errors()[0].Summary(), diags.Errors()[0].Detail())
	}
	newID.strings = elements
	return &newID, nil
}

func (id txtRecordID) String() string {
	strs := make([]string, len(id.strings))
	for i, s := range id.strings {
		strs[i] = url.QueryEscape(s)
	}
	return fmt.Sprintf("%s/%s/%s/%s", id.zone, id.view, id.name, strings.Join(strs, ","))
}

func (id txtRecordID) getFqdn() string {
	if name := id.name; strings.HasSuffix(name, ".") {
		return name
	} else {
		return fmt.Sprintf("%s.%s.", name, id.zone)
	}
}

// copyIDToModel set attributes in the model to values
// from which the ID was composed
func (r txtRecordResource) restoreIDAttributesToModel(ctx context.Context, id txtRecordID, rm *txtRecordResourceModel) error {
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
	var diags diag.Diagnostics
	rm.Strings, diags = types.ListValueFrom(ctx, types.StringType, id.strings)
	if diags.HasError() {
		return fmt.Errorf("error parsing strings from ID, the first error: %s (%s)", diags.Errors()[0].Summary(), diags.Errors()[0].Detail())
	}
	return nil
}

// readInDimResponse parse the DIM response into the Resource Model
func (r txtRecordResource) readInDimResponse(dimResp map[string]any, rm *txtRecordResourceModel) {
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

func (r *txtRecordResource) diagErrorSummaryTemplate() string {
	return "Error in %s TXT record"
}

func (r *txtRecordResource) diagErrorDetailTemplate() string {
	return "Unexpected error from %s: %s"
}

// dimRawCall is a helper function to call DIM API and handle errors and logging
func (r *txtRecordResource) dimRawCall(ctx context.Context, tfAction string, dfunc string, dargs []any, diags *diag.Diagnostics) (any, error) {
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
func (r *txtRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *txtRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_txt_record"
}

// Schema defines the schema for the resource.
func (r *txtRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a TXT type record in DIM.",
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

			"strings": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
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
func (r *txtRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Create a new resource.
	// Retrieve values from data
	var data txtRecordResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// required args
	// prepare TXT record strings
	id, err := newTxtRecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Create"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
		return
	}
	dim_req_args := map[string]any{
		"type":    "TXT",
		"name":    id.name,
		"strings": id.strings,
	}
	// required args: strings
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

	dim_req_args["name"] = id.getFqdn() // rr_get_attrs has no "zone" arg, so "name" arg must be fqdn
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
func (r *txtRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current data
	var data txtRecordResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newTxtRecordIDFromString(data.ID.ValueString())
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
		"type":    "TXT",
		"name":    id.getFqdn(), // rr_set_attrs accepts no "zone" arg, so "name" must be fqdn with trailing dot
		"strings": id.strings,
	}
	// optional args
	// we do not check if !data.View.IsNull(), because the state might be null
	// , e.g, after import
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
func (r *txtRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// only TTL and comment are updatable
	// other args result in resource replacement

	// Retrieve values from data
	var data txtRecordResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// required args
	id, err := newTxtRecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Update"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
		return
	}
	dim_req_args := map[string]any{
		"type":    "TXT",
		"name":    id.getFqdn(), // rr_set_attrs has no "zone" attr, so "name" must be fqdn with trailing dot
		"strings": id.strings,
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
func (r *txtRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from data
	var data txtRecordResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := newTxtRecordIDFromTfModel(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
			fmt.Sprintf("Unable to compose ID from Resource Model: %s", err.Error()),
		)
		return
	}

	// required args
	dim_req_args := map[string]any{
		"type":    "TXT",
		"name":    id.name,
		"strings": id.strings,
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

func (r *txtRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
