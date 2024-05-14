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

	Comment types.String `tfsdk:"comment"`
}

type ipID struct {
	layer3domain string
	ip           string
}

func (rm ipResourceModel) composeID() string {
	// <layer3domain>/<ip>
	return rm.Layer3domain.ValueString() + "/" + rm.Ip.ValueString()
}

func (rm ipResourceModel) parseID() (ipID, error) {
	idParts := strings.SplitN(rm.ID.ValueString(), "/", 2)
	if len(idParts) != 2 {
		return ipID{}, fmt.Errorf("ID is not in expected format")
	}
	return ipID{
		layer3domain: idParts[0],
		ip:           idParts[1],
	}, nil
}

func (rm *ipResourceModel) readInDimResponse(dimResp map[string]any) {
	if v, ok := dimResp["ip"]; ok {
		rm.Ip = types.StringValue(v.(string))
	}
	if v, ok := dimResp["layer3domain"]; ok {
		rm.Layer3domain = types.StringValue(v.(string))
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

	if v, ok := dimResp["comment"]; ok {
		rm.Comment = types.StringValue(v.(string))
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
	tflog.Debug(ctx, fmt.Sprintf("ip/%s dim-call", tfAction), map[string]any{"func": dfunc, "args": dargs})
	dimResp, err := r.client.RawCallWithContext(ctx, dfunc, dargs)
	if err != nil {
		diags.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), tfAction),
			fmt.Sprintf(r.diagErrorDetailTemplate(), dfunc, err.Error()),
		)
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("ip/%s dim-response", tfAction), map[string]any{"dimResponse": dimResp})
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
		MarkdownDescription: "Allocates an ip address from the pool (i.e. set `status` to `Static`).\n" +
			" - If the `ip` argument left unspecified," +
			" it will allocate the next free (`status` = `Available`) ip address from the pool;\n" +
			" - If `ip` is specified, it must be free (`status` = `Available` ) upon resource creation.",
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
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "If specified, this address will be allocated. The address must be within the `pool` specified. If not set, an available address will be allocated from the pool.",
			},
			"pool": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "The pool where the IP address is allocated.",
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The comment to the allocated IP address",
			},

			"layer3domain": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The layer 3 domain where the IP address is allocated.",
			},
			"created": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mask": schema.StringAttribute{
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
			"reverse_zone": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "the known status values are:\n" +
					"  - `Static` a single allocated IP address\n" +
					"  - `Available` a single free IP address\n" +
					"  - `Reserved` a reserved single IP address (for example the IPv4 network and broadcast addresses in a Subnet)\n" +
					"  - `Container` a generic status for blocks larger than subnets\n" +
					"  - `Delegation` the block is used for a specific purpose (ex: a server)\n" +
					"  - `Subnet` a subnet (can only have Delegation, Static, Reserved or Available children)",
			},
			"subnet": schema.StringAttribute{
				//Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	// Create a new resource.
	// Retrieve values from data
	var data ipResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dim_req_named_args := map[string]any{}

	if !data.Comment.IsNull() {
		dim_req_named_args["attributes"] = map[string]any{"comment": data.Comment.ValueString()}
	}

	var dimResp any

	if data.Ip.IsUnknown() {
		// will get a free IP from the pool
		dimResp, _ = r.dimRawCall(ctx, "Create",
			"ippool_get_ip",
			[]any{
				data.Pool.ValueString(),
				dim_req_named_args,
			},
			&resp.Diagnostics,
		)
	} else {
		// will reserve the specific IP checking:
		// 1) that it's within the pool
		// 2) that it's a host
		// 3) although doc specifies that status might be checked,
		//    specifying the status returns error "ip_mark error (19): Unknown options: status"
		//    so it's not set here. But ip_mark refuses to make Static the addresss
		//    which is alrady static, so we are safe.
		dim_req_named_args["pool"] = data.Pool.ValueString()
		dim_req_named_args["host"] = true
		dimResp, _ = r.dimRawCall(ctx, "Create",
			"ip_mark",
			[]any{
				data.Ip.ValueString(),
				dim_req_named_args,
			},
			&resp.Diagnostics,
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	data.readInDimResponse(dimResp.(map[string]any))
	tflog.Info(ctx, "IP has been made static", map[string]any{
		"layer3domain": data.Layer3domain.ValueString(),
		"ip":           data.Ip.ValueString(),
	})
	// now when we know the all values, set the ID
	data.ID = types.StringValue(data.composeID())

	// Set state to fully populated data
	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *ipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current data
	var data ipResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := data.parseID()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("ID parsed %+v", id))

	dim_req_args := map[string]any{
		"host":         true,
		"layer3domain": id.layer3domain,
	}

	dimResp, _ := r.dimRawCall(ctx, "Read",
		"ipblock_get_attrs",
		[]any{
			id.ip,
			dim_req_args,
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	data.readInDimResponse(dimResp.(map[string]any))

	if data.ID.ValueString() != data.composeID() {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
			fmt.Sprintf("ID has changed, old=%s, new=%s", data.ID.ValueString(), data.composeID()),
		)
		return
	}

	// terraform recommends call State.RemoveResource if the resource no longer exists, see:
	// https://developer.hashicorp.com/terraform/plugin/framework/resources/read#recommendations
	//
	// originally:
	// if data.Status.ValueString() != "Static" {
	// 	resp.Diagnostics.AddError(
	// 		fmt.Sprintf(r.diagErrorSummaryTemplate(), "Read"),
	// 		"IP is not in Static state",
	// 	)
	// 	return
	// }
	if data.Status.ValueString() != "Static" {
		tflog.Debug(ctx, fmt.Sprintf("the status of the IP is not Static (has been released?) %+v", id))
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "IP has been read", map[string]any{
		"layer3domain": data.Layer3domain.ValueString(),
		"ip":           data.Ip.ValueString(),
	})
	// Set refreshed state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data ipResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, _ := data.parseID() // well we know it's valid, so no need to check the error

	opts := map[string]any{
		"host":         true,
		"layer3domain": id.layer3domain,
		"pool":         data.Pool.ValueString(),
	}

	_, _ = r.dimRawCall(ctx, "Update",
		"ipblock_set_attrs",
		[]any{
			id.ip,
			// attributes
			map[string]any{
				"comment": data.Comment.ValueString(),
			},
			// options
			opts,
		}, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IP has been updated", map[string]any{
		"layer3domain": id.layer3domain,
		"ip":           id.ip,
	})

	// Read the updated attrs

	dimResp, _ := r.dimRawCall(ctx, "Update",
		"ipblock_get_attrs",
		[]any{
			id.ip,
			opts,
		},
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	data.readInDimResponse(dimResp.(map[string]any))

	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from data
	var data ipResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := data.parseID()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(r.diagErrorSummaryTemplate(), "Delete"),
			err.Error(),
		)
		return
	}

	dim_req_args := map[string]any{
		"layer3domain": id.layer3domain,
		"host":         true,
		"pool":         data.Pool.ValueString(),
	}

	dimResp, _ := r.dimRawCall(ctx, "Delete",
		"ip_free",
		[]any{
			id.ip,
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
