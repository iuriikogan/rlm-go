gcloud iam workload-identity-pools create "" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --display-name="Demo pool"

gcloud iam workload-identity-pools providers create-oidc "gha-provider" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --workload-identity-pool="gha-pool" \
  --display-name="gha-provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.aud=assertion.aud" \
  --issuer-uri="https://token.actions.githubusercontent.com"
