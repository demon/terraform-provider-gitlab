package provider

import (
	"context"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "github.com/xanzy/go-gitlab"
)

var _ = registerResource("gitlab_instance_variable", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_instance_variable` + "`" + ` resource allows to manage the lifecycle of a CI/CD variable for an instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/instance_level_variables.html)`,

		CreateContext: resourceGitlabInstanceVariableCreate,
		ReadContext:   resourceGitlabInstanceVariableRead,
		UpdateContext: resourceGitlabInstanceVariableUpdate,
		DeleteContext: resourceGitlabInstanceVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"key": {
				Description:  "The name of the variable.",
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: StringIsGitlabVariableName,
			},
			"value": {
				Description: "The value of the variable.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"variable_type": {
				Description:  "The type of a variable. Available types are: env_var (default) and file.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "env_var",
				ValidateFunc: StringIsGitlabVariableType,
			},
			"protected": {
				Description: "If set to `true`, the variable will be passed only to pipelines running on protected branches and tags. Defaults to `false`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"masked": {
				Description: "If set to `true`, the value of the variable will be hidden in job logs. The value must meet the [masking requirements](https://docs.gitlab.com/ee/ci/variables/#masked-variable-requirements). Defaults to `false`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
})

func resourceGitlabInstanceVariableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)

	options := gitlab.CreateInstanceVariableOptions{
		Key:          &key,
		Value:        &value,
		VariableType: variableType,
		Protected:    &protected,
		Masked:       &masked,
	}
	log.Printf("[DEBUG] create gitlab instance level CI variable %s", key)

	_, _, err := client.InstanceVariables.CreateVariable(&options, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(d, err)
	}

	d.SetId(key)

	return resourceGitlabInstanceVariableRead(ctx, d, meta)
}

func resourceGitlabInstanceVariableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	key := d.Id()

	log.Printf("[DEBUG] read gitlab instance level CI variable %s", key)

	v, resp, err := client.InstanceVariables.GetVariable(key, gitlab.WithContext(ctx))
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[DEBUG] gitlab instance level CI variable for %s not found so removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return augmentVariableClientError(d, err)
	}

	d.Set("key", v.Key)
	d.Set("value", v.Value)
	d.Set("variable_type", v.VariableType)
	d.Set("protected", v.Protected)
	d.Set("masked", v.Masked)
	return nil
}

func resourceGitlabInstanceVariableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	key := d.Get("key").(string)
	value := d.Get("value").(string)
	variableType := stringToVariableType(d.Get("variable_type").(string))
	protected := d.Get("protected").(bool)
	masked := d.Get("masked").(bool)

	options := &gitlab.UpdateInstanceVariableOptions{
		Value:        &value,
		Protected:    &protected,
		VariableType: variableType,
		Masked:       &masked,
	}
	log.Printf("[DEBUG] update gitlab instance level CI variable %s", key)

	_, _, err := client.InstanceVariables.UpdateVariable(key, options, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(d, err)
	}
	return resourceGitlabInstanceVariableRead(ctx, d, meta)
}

func resourceGitlabInstanceVariableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	key := d.Get("key").(string)
	log.Printf("[DEBUG] Delete gitlab instance level CI variable %s", key)

	_, err := client.InstanceVariables.RemoveVariable(key, gitlab.WithContext(ctx))
	if err != nil {
		return augmentVariableClientError(d, err)
	}

	return nil
}
