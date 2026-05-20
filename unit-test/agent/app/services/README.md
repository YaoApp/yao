# Services Directory

This directory is scanned by `script.Load` to load service scripts.

It is intentionally kept empty for the test application.
The directory must exist to prevent `GetFileAttributesEx` errors on Windows
when `script.Load` attempts to stat it during test initialization.
