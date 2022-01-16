# Umami Pulumi

## The goal
Create two virtual-machines for master-worker Kubernetes nodes in GCP. Only allow connections to ports 30000-40000 from a single IP address.

## Pulumi configuration
You will need to setup custom configuration variables to be able to run the project.

Example:
```yaml
  umami:ip-cidr: 0.0.0.0/0
  umami:project: project-id
  umami:region: europe-west1
  umami:ssh-pubkey-path: /path/to/public/ssh/key
  umami:zone: europe-west1-a
```
