// Package cloud_run contains deployment artifacts for Google Cloud Run.
// Dockerfile and related configs live here.
// Build context must be the project root (LineBot/Backend/).
//
// Build:
//   docker build -f deployee/cloud_run/Dockerfile .
//
// Deploy:
//   gcloud run deploy linebot --source . --region asia-east1
package cloud_run
