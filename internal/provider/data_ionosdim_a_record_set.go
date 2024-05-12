package provider

import (
	"context"
	"fmt"

	"terraform-provider-ionosdim/pkg/dim"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &aRecordSetDataSource{}
	_ datasource.DataSourceWithConfigure = &aRecordSetDataSource{}
)

// NewCoffeesDataSource is a helper function to simplify the provider implementation.
func NewARecordSetDataSource() datasource.DataSource {
	return &aRecordSetDataSource{}
}

// coffeesDataSource is the data source implementation.
type aRecordSetDataSource struct {
	client *dim.Client
}

// data model
type aRecordSetDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Host         types.String `tfsdk:"host"`
	Addrs        types.List   `tfsdk:"addrs"`
	Layer3domain types.String `tfsdk:"layer3domain"`
	Zone         types.String `tfsdk:"zone"`
	View         types.String `tfsdk:"view"`
	TTL          types.Number `tfsdk:"ttl"`
}

// Metadata returns the data source type name.
func (d *aRecordSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_a_record_set"
}

// Schema defines the schema for the data source.
func (d *aRecordSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get DNS A records of the host.",
		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				Required:    true,
				Description: "Zone.",
			},
			"view": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "View.",
			},
			"layer3domain": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "layer3domain.",
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Host to look up.",
			},
			"addrs": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "A list of IP addresses. IP addresses are always sorted to avoid constant changing plans.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Always set to the host.",
			},
			"ttl": schema.NumberAttribute{
				Computed:    true,
				Description: "ttl.",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *aRecordSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config aRecordSetDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dim_req_args := map[string]interface{}{"type": "A", "pattern": config.Host.ValueString()}

	if !config.Layer3domain.IsNull() {
		dim_req_args["layer3domain"] = config.Layer3domain.ValueString()
	}
	if !config.View.IsNull() {
		dim_req_args["view"] = config.View.ValueString()
	}
	if !config.Zone.IsNull() {
		dim_req_args["zone"] = config.Zone.ValueString()
	}

	host := config.Host.ValueString()
	dimRes, err := d.client.RawCallWithContext(ctx, "rr_list", []interface{}{dim_req_args})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("error looking up A records for %q: ", host), err.Error())
		return
	}

	addrs := []string{}
	var view, layer3domain []string
	//var ttl []int

	for _, rr := range interface{}(dimRes).([]interface{}) {
		addrs = append(addrs, (rr.(map[string]interface{}))["value"].(string))
		view = append(view, (rr.(map[string]interface{}))["view"].(string))
		layer3domain = append(layer3domain, (rr.(map[string]interface{}))["layer3domain"].(string))
	}
	//sort.Strings(addrs)

	//voodoo
	var convertDiags diag.Diagnostics
	config.Addrs, convertDiags = types.ListValueFrom(ctx, config.Addrs.ElementType(ctx), addrs)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = config.Host

	if length := len(addrs); length > 0 {
		// all view and layer3domain should be the same
		// add diag if view or layer3domain are different
		if view[0] != view[length-1] {
			resp.Diagnostics.AddWarning(
				"Multiple Views",
				fmt.Sprintf("Multiple views found: %v", view),
			)
		}
		if layer3domain[0] != layer3domain[length-1] {
			resp.Diagnostics.AddWarning(
				"Multiple Layer3domains",
				fmt.Sprintf("Multiple layer3domains found: %v", layer3domain),
			)
		}

		config.View = types.StringValue(view[0])
		config.Layer3domain = types.StringValue(layer3domain[0])
	}

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Configure adds the provider configured client to the data source.
func (d *aRecordSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}
