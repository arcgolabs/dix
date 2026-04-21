---
title: 'error-returning providers'
linkTitle: 'error providers'
description: 'Use Err-suffixed dix helpers when provider construction can fail'
weight: 25
---

## Error-returning providers

Use the `Err` suffixed APIs when service construction can fail and you want that failure to propagate through normal `dix` resolution.

## Why a separate API

Go does not support overloading by return type. That means `Provider0(func() T)` cannot also accept `func() (T, error)` without making the API ambiguous.

`dix` therefore keeps the existing APIs unchanged and adds explicit `Err` variants.

## Core APIs

- `dix.ProviderErr0..6`
- `dixadvanced.NamedProviderErr0..3`
- `dixadvanced.TransientProviderErr0..1`
- `dixadvanced.NamedTransientProviderErr0..1`

## Scoped and override APIs

- `dixadvanced.ProvideScopedErr0..3`
- `dixadvanced.ProvideScopedNamedErr0..3`
- `dixadvanced.OverrideErr0..1`
- `dixadvanced.NamedOverrideErr0..1`
- `dixadvanced.OverrideTransientErr0..1`
- `dixadvanced.NamedOverrideTransientErr0..1`

## Example

```go
app := dix.NewApp("app",
    dix.NewModule("infra",
        dix.WithModuleProviders(
            dix.ProviderErr1(func(cfg Config) (*DB, error) {
                return OpenDB(cfg.DSN)
            }),
        ),
    ),
)
```

## Scoped example

```go
scope := advanced.Scope(rt, "request-42", func(injector do.Injector) {
    advanced.ProvideScopedNamedErr0(injector, "tenant.default", func() (string, error) {
        return resolveTenantFromRequest()
    })
})
```

## Override example

```go
module := dix.NewModule("test",
    dix.WithModuleSetups(
        advanced.OverrideErr0(func() (*Config, error) {
            return loadFixtureConfig()
        }),
    ),
)
```

## Rule of thumb

- Use `ProviderN` when construction is pure and cannot fail.
- Use `ProviderErrN` when construction performs I/O, parsing, validation, or any other fallible step.
- Keep the `Err` suffix through `advanced` helpers so call sites remain explicit.
