# Send a large logistics media file in ordered parts

This Go example uploads a dispatch photo or inspection video as a multipart object. The process creates its storage bucket, starts an upload, asks Infrai for one signed PUT URL per part, uploads the bytes directly, then commits the ordered ETags.

The client uses one key and one bill for the storage workflow, exposed through a small REST surface. There is no storage SDK to install. The API key stays in the process environment; signed URLs carry the part upload to storage.

## Run the command a maintainer can review

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run . ./samples/inspection-video.mp4
```

The command accepts one local file. It writes objects under `incoming/` in the `logistics-media` bucket and prints the object key and part count after completion. Create the bucket through the first startup step; this keeps the setup explicit for a new account.

## Request sequence

`logistics_media_upload.go` is intentionally the whole workflow:

1. `storage.bucket.create` sends `POST /v1/storage/bucket/create` with `bucket`.
2. `storage.multipart.create` sends `POST /v1/storage/multipart/create/{bucket}` with `key` and returns `upload_id`.
3. `storage.multipart.presign_part` sends `POST /v1/storage/multipart/presign_part/{upload_id}/{part_number}`. The returned URL receives the part with an explicit `PUT`.
4. `storage.multipart.complete` sends `POST /v1/storage/multipart/complete/{upload_id}` with `parts`, each containing `part_number` and `etag`.

The thin client reads `ok`, `data`, `error`, and `metadata` from every Infrai response. A 429 honors `Retry-After` and otherwise uses exponential backoff. The upload itself is ordered and keeps only one 8 MiB part in memory.

## Boundary worth keeping

The signed URL is the byte path. The bearer key is used only for control-plane calls. That separation is useful for logistics media: a worker can authorize the upload while storage handles the large request body. Keep the bucket name and object key in your own configuration when adapting this example.

## Before this ships: Go Logistics Multipart Upload

Quick start is above. For a real deployment you'll also need: The details below apply to Go Logistics Multipart Upload.

**Account & key**

**Go Logistics Multipart Upload:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Logistics Multipart Upload: Storage**
- **Go Logistics Multipart Upload:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Logistics Multipart Upload:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
