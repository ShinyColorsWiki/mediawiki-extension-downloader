# Mediawiki Extension Downloader
(Will) used for our wiki extension downloader during image build.

# Usage
```bash
$ mediawiki-extension-downloader --config ./config.json --target ./extension --force-rm-target=true
```
See `config.json.example` to how to configure

# Docker
[`ghcr.io/ShinyColorsWiki/mediawiki-extension-downloader`](https://ghcr.io/ShinyColorsWiki/mediawiki-extension-downloader)

# Limits/Known Issues
* WMF extensions are downloaded using github. Will use Their GitLab once migrated all.
* The `Git` actually doesn't use the git. Only supports GitHub and GitLab now.
* This program will force to `strip-components=1`. This may has issue on `http` extensions. \

# Todo
* Fix known issues.
* Support download skin.