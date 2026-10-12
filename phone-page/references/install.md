# Install phone-page

## Local file

The complete format of `~/.agents/local/phone-page.md` is one line containing an executable name from `PATH` or an absolute executable path:

```text
page-store
```

`PHONE_PAGE_LOCAL_FILE` overrides the path for tests. The command must implement the [store contract](../../cf-page-store/references/store-contract.md) that the cf-page-store skill defines.

## Steps

1. Make `phone-page/phone-page` executable.
2. Link `phone-page/phone-page` into a directory on `PATH`, such as `~/.local/bin`.
3. Link the `phone-page` directory from the checkout into every agent CLI's skills directory, then verify that each CLI lists the skill.
4. Create the local file with the chosen store command after that store is installed.
5. Run `phone-page assets <directory>` once for the store and confirm that the store serves every downloaded file at `/assets/<name>`.
