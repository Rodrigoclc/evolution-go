# Rodrigo Castro's Evolution GO fork

This repository is a maintenance fork of `evolution-foundation/evolution-go`, based on upstream `0.7.2`.

## Purpose

The fork exists to carry narrowly scoped operational fixes required by Rodrigo's services while upstream releases are unavailable:

- bounded and shared PostgreSQL pools for WhatsApp device storage;
- observable and testable WhatsApp Web version resolution;
- regression coverage for QR pairing and reconnection;
- on-demand full label resync (`POST /label/sync`), so labels and labeled chats that existed before pairing reach the label table and the webhook, with the phone-number JID (`JIDAlt`) for LID-addressed chats;
- immutable images published to `ghcr.io/rodrigoclc/evolution-go`.

## Upstream synchronization

- `origin`: Rodrigo's fork;
- `upstream`: official Evolution Foundation repository;
- production uses only `sha-<commit>` tags or image digests;
- upstream updates are merged separately from local patches and must pass the complete CI matrix.

## Trademark and attribution

This fork is not an official Evolution Foundation release. Evolution GO and its upstream source remain attributed according to `LICENSE`, `NOTICE`, and `TRADEMARKS.md`. Any customized user interface or distribution must use distinct branding and preserve required notices.
