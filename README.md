# terraform-provider-plexsphere

Terraform provider for the [plexsphere](https://plexsphere.com) v1 API.

The provider is built on the [terraform-plugin-framework][tpf] and follows a
**spec-driven** workflow that mirrors [`plexsphere-sdk-generator`][gen]: a single
OpenAPI document is the source of truth, and as much code as possible is
generated from it.

## How code is generated

```
spec/plexsphere-v1.yaml  (vendored, checksummed — same source as the SDKs)
        │
        │  tfplugingen-openapi   (HashiCorp terraform-plugin-codegen-openapi)
        ▼
provider-code-spec.json  (Provider Code Specification)
        │
        │  tfplugingen-framework (HashiCorp terraform-plugin-codegen-framework)
        ▼
internal/provider/*_gen.go        (resource schema + model — DO NOT EDIT)
internal/datasources/*_gen.go     (data-source schema + model — DO NOT EDIT)
        │
        │  + hand-written CRUD glue calling plexsphere-sdk-go
        ▼
a working provider
```

The codegen tools deliberately generate **schema and models only** — never an
API client and never CRUD logic. The HTTP calls are made through the existing
[`plexsphere-sdk-go`][sdk]; the hand-written layer maps between the Terraform
model and the SDK model. That mapping is the one seam that stays hand-written
(see [`internal/provider/project_resource.go`](internal/provider/project_resource.go)).

What is generated vs. hand-written:

| Concern                                   | Source                              |
| ----------------------------------------- | ----------------------------------- |
| Attribute names, types, validators        | generated (`*_gen.go`)              |
| `Create`/`Read`/`Update`/`Delete` logic   | hand-written                        |
| TF-model ⇄ SDK-model mapping              | hand-written (`flattenProject`)     |
| Plan modifiers (`RequiresReplace`, etc.)  | hand-written overlay in `Schema()`  |
| HTTP client + request/response types      | [`plexsphere-sdk-go`][sdk]          |

Plan modifiers cannot be expressed by the generator, so they are applied as a
small overlay on top of the generated schema in the resource's `Schema()` method
— the generated file is never edited and can be regenerated at any time.

## Regenerating after a spec change

```sh
make fetch-spec   # pull + checksum the latest spec/plexsphere-v1.yaml
make generate     # spec -> provider-code-spec.json -> *_gen.go, then gofmt
make build        # compile
```

`make generate` is equivalent to `go generate ./...` (see the directives in
[`main.go`](main.go)). The codegen tools are pinned in
[`tools/tools.go`](tools/tools.go).

Which resources/data sources are generated is controlled by
[`generator_config.yml`](generator_config.yml) — each entry maps a Terraform
resource onto the spec's CRUD operations by path + method.

## Pilot scope

The current pilot covers a single resource and data source — `plexsphere_project`
(tenancy) — chosen because it exercises the full pattern: required vs.
optional-computed attributes, server-assigned computed fields (`id`,
`created_at`, `updated_at`), immutable attributes (`domain_id`, `slug` →
`RequiresReplace`), and a real foreign key. Adding the next resource is mostly:
extend `generator_config.yml`, run `make generate`, write its CRUD glue.

> **Multi-package note:** `tfplugingen-framework` names every model `<Name>Model`,
> so generating a resource *and* a data source for the same entity into one Go
> package collides on `ProjectModel`. Data sources are therefore generated into
> their own package (`internal/datasources`). Scale this by giving each
> resource/data-source group its own package.

## Using the provider locally

```hcl
provider "plexsphere" {
  endpoint = "https://api.plexsphere.com" # or PLEXSPHERE_ENDPOINT
  # token via PLEXSPHERE_TOKEN
}
```

Authentication uses a bearer token (`Authorization: Bearer …`), which the spec
exempts from its CSRF handshake. See [`examples/`](examples/).

[tpf]: https://developer.hashicorp.com/terraform/plugin/framework
[gen]: https://github.com/plexsphere/plexsphere-sdk-generator
[sdk]: https://github.com/plexsphere/plexsphere-sdk-go
