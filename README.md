# Mediawiki Extension Downloader
Used for our wiki extension downloader during image build.

# Usage
```bash
$ mediawiki-extension-downloader --config ./config.json --target ./extension --force-rm-target=true
```
See `config.example.json` to how to configure. \
Note: Environment variable `MWREL` has high priority than config's `MWREL` settings.

For `Git` entries, set either `branch` or `tag`. If neither is set, the downloader uses `MWREL` as the branch.

```json
{
  "Extensions": {
    "Git": {
      "ExampleFromBranch": {
        "type": "github",
        "repo": "example/mediawiki-extension-example",
        "branch": "REL1_43"
      },
      "ExampleFromTag": {
        "type": "github",
        "repo": "example/mediawiki-extension-example",
        "tag": "v1.2.3"
      }
    }
  }
}
```

# Docker
[`ghcr.io/ShinyColorsWiki/mediawiki-extension-downloader`](https://ghcr.io/ShinyColorsWiki/mediawiki-extension-downloader)

# Limits/Known Issues
* WMF extensions are downloaded via ExtDist with Gerrit. Maybe this will be changed future due to upstream.
* The `Git` actually doesn't use the git. Only supports GitHub and GitLab now via download from the site itself.
* This program will force to `strip-components=1`. This may has issue on `http` extensions.

# Todo
* Fix known issues.
