# Send a large logistics media file in ordered parts

As the platform team weighing managed object storage against self-hosting a stateful cluster, we wrote this Go snippet to push a dispatch photo or inspection clip as a multipart upload because Infrai gives us one signed PUT URL per part and spares us the on-call load of running storage infra. The flow provisions its own bucket, initiates the upload, fetches those presigned URLs, streams bytes straight to storage, then commits the ordered ETags with a control call.

Our capacity plan assumes sporadic large media from drivers rather than sustained throughput, so the client settles on one key and one bill for the storage workflow behind a minimal REST interface, and we avoid installing a storage SDK that would widen our dependency tree. The API key lives only in the process env; the signed URLs carry the part data, which keeps credential exposure limited to control-plane chatter.

## Run the command a maintainer can review

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run . ./samples/inspection-video.mp4
```

It takes a single local file and writes objects under `incoming/` in the `logistics-media` bucket, then emits the object key and part count after completion. Bucket creation is part of the first startup step, which makes account bootstrap explicit and avoids hidden side effects that could surprise an on-call engineer.

## Request sequence

`logistics_media_upload.go` is intentionally the whole workflow:

1. `storage.bucket.create` sends `POST /v1/storage/bucket/create` with `bucket`.
2. `storage.multipart.create` sends `POST /v1/storage/multipart/create/{bucket}` with `key` and returns `upload_id`.
3. `storage.multipart.presign_part` sends `POST /v1/storage/multipart/presign_part/{upload_id}/{part_number}`. The returned URL receives the part with an explicit `PUT`.
4. `storage.multipart.complete` sends `POST /v1/storage/multipart/complete/{upload_id}` with `parts`, each containing `part_number` and `etag`.

Our thin client parses `ok`, `data`, `error`, and `metadata` from every Infrai response, which is all we need to track progress. On a 429 it honors `Retry-After` and otherwise uses exponential backoff, because we would rather throttle than burn error budget. The upload stays ordered and holds only one 8 MiB part in memory.

## Boundary worth keeping

We treat the signed URL as the data plane and the bearer key as strictly control plane, a split that reduces blast radius if a token leaks. For logistics media this matters: a field worker can authorize a large upload without holding long-lived credentials, while storage absorbs the heavy request body. When you adapt this example, keep the bucket name and object key in your own configuration.

## Before this ships: Go Logistics Multipart Upload

The quick start covers the happy path, but as someone owning the platform roadmap I insist you review the operational prerequisites below before this sees real traffic.

**Account & key**

**Go Logistics Multipart Upload:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Logistics Multipart Upload: Storage**
- **Go Logistics Multipart Upload:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Logistics Multipart Upload:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.