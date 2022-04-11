# Project and Application Labels

When creating a new Project, it is possible to supply labels to the resulting ApplicationSet template. These templates will end up on the generated Applications of that Project.

## Using static labels
The simplest and most straight-forward way of using such templates is to supply "key=value" pairs
when creating a new Project:

```shell
argocd-autopilot project create my-proj --labels "app.my.org/name=org-name","app.my.org/type=org-type"
```

this will add those labels to the ApplicationSet template, and they will end up as-is on every Application generated from it.

## Using dynamic labels

A more interesting usage of this flag is to supply dynamic labels that will be populated by different
value

```shell
argocd-autopilot project create my-proj --labels "app.my.org/dynamic-label={{ app_my_org_dynamic_label }}"
```

This will add the label to the template, with the placeholder of `{{ app_my_org_dynamic_label }}` as the label's value.  
Then create **all** applications in that Project with:

```shell
# creating "my-app" in "my-proj"
argocd-autopilot app create my-app --app github.com/... --project my-proj --labels "app_my_org_dynamic_label=specific-value-my-app"

# creating "different-app" in "my-proj"
argocd-autopilot app create different-app --app github.com/... --project my-proj --labels "app_my_org_dynamic_label=specific-value-different-app"
```

This will put the different values inside each app's `config.json` file, which will later be used by the ApplicationSet during the Application generation to replace the placeholder strings in the template.

### Caveats

It is imprortant to note that creating a Project with dynamic labels **requires** that all following `app create` calls will be made with matching values to replace the original placeholder string. Failing to do so will cause the ApplicationSet to fail in generating the Application, and might also effect other applications in the same Project.