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
	_ datasource.DataSource              = &cnameRecordSetDataSource{}
	_ datasource.DataSourceWithConfigure = &cnameRecordSetDataSource{}
)

// NewCoffeesDataSource is a helper function to simplify the provider implementation.
func NewCNAMERecordSetDataSource() datasource.DataSource {
	return &cnameRecordSetDataSource{}
}

// coffeesDataSource is the data source implementation.
type cnameRecordSetDataSource struct {
	client *dim.Client
}

// data model
type cnameRecordSetDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Host  types.String `tfsdk:"host"`
	Addrs types.List   `tfsdk:"addrs"`
}

// Metadata returns the data source type name.
func (d *cnameRecordSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cname_record_set"
}

// Schema defines the schema for the data source.
func (d *cnameRecordSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get DNS A records of the host.",
		Attributes: map[string]schema.Attribute{
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
			// "ttl": schema.NumberAttribute{
			// 	Computed:    true,
			// 	Description: "TTL of the record",
			// },
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *cnameRecordSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state cnameRecordSetDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := state.Host.ValueString()
	dimRes, err := d.client.RawCall("rr_list", []interface{}{map[string]interface{}{
		"type":    "CNAME",
		"pattern": host,
		"fields":  true,
	}})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("error looking up CNAME records for %q: ", host), err.Error())
		return
	}

	addrs := []string{}
	for _, rr := range interface{}(dimRes).([]interface{}) {
		addrs = append(addrs, (rr.(map[string]interface{}))["value"].(string))
	}
	//sort.Strings(addrs)

	//voodoo
	var convertDiags diag.Diagnostics
	state.Addrs, convertDiags = types.ListValueFrom(ctx, state.Addrs.ElementType(ctx), addrs)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = state.Host

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Configure adds the provider configured client to the data source.
func (d *cnameRecordSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
