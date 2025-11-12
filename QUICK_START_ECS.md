# Quick Start: Deploy to Your ECS Cluster

Your ECS cluster is already created:
- **Cluster ARN**: `arn:aws:ecs:us-west-2:296921411200:cluster/unique-zebra-ow4l1t`
- **Cluster Name**: `unique-zebra-ow4l1t`
- **Region**: `us-west-2`
- **Account ID**: `296921411200`

## Step 1: Create ECR Repository

```bash
aws ecr create-repository \
  --repository-name startup-dose-api \
  --region us-west-2
```

## Step 2: Run the Deployment Script

```bash
./scripts/deploy-ecs.sh
```

The script will:
1. Build the Docker image
2. Push it to ECR
3. Register the task definition
4. Create/update the ECS service

## Step 3: Verify Deployment

Check the service status:
```bash
aws ecs describe-services \
  --cluster unique-zebra-ow4l1t \
  --services startup-dose-api-service \
  --region us-west-2
```

View task logs:
```bash
aws logs tail /ecs/startup-dose-api --follow --region us-west-2
```

## Manual Setup (Alternative)

If you prefer to set up manually:

### 1. Login to ECR
```bash
aws ecr get-login-password --region us-west-2 | \
  docker login --username AWS --password-stdin 296921411200.dkr.ecr.us-west-2.amazonaws.com
```

### 2. Build and Push Image
```bash
docker build -t startup-dose-api:latest .
docker tag startup-dose-api:latest 296921411200.dkr.ecr.us-west-2.amazonaws.com/startup-dose-api:latest
docker push 296921411200.dkr.ecr.us-west-2.amazonaws.com/startup-dose-api:latest
```

### 3. Register Task Definition
```bash
aws ecs register-task-definition \
  --family startup-dose-api \
  --network-mode awsvpc \
  --requires-compatibilities FARGATE \
  --cpu 256 \
  --memory 512 \
  --cli-input-json file://ecs-task-definition.json \
  --region us-west-2
```

### 4. Create ECS Service
```bash
aws ecs create-service \
  --cluster unique-zebra-ow4l1t \
  --service-name startup-dose-api-service \
  --task-definition startup-dose-api:1 \
  --desired-count 1 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxxx],securityGroups=[sg-xxxxx]}" \
  --region us-west-2
```

**Note**: You'll need to replace `subnet-xxxxx` and `sg-xxxxx` with your actual subnet and security group IDs.

## View in AWS Console

Open the AWS ECS console to monitor your cluster:
```
https://us-west-2.console.aws.amazon.com/ecs/v2/clusters/unique-zebra-ow4l1t
```

## Environment Variables

To customize the deployment, you can set:

```bash
export AWS_REGION=us-west-2
export AWS_ACCOUNT_ID=296921411200
export ECS_CLUSTER_NAME=unique-zebra-ow4l1t
export ECS_SERVICE_NAME=startup-dose-api-service
export IMAGE_TAG=latest

./scripts/deploy-ecs.sh
```

## Troubleshooting

### Service won't reach desired count
1. Check task definition is valid:
   ```bash
   aws ecs describe-task-definition \
     --task-definition startup-dose-api \
     --region us-west-2
   ```

2. Check task logs:
   ```bash
   aws ecs list-tasks \
     --cluster unique-zebra-ow4l1t \
     --region us-west-2

   aws ecs describe-tasks \
     --cluster unique-zebra-ow4l1t \
     --tasks <task-arn> \
     --region us-west-2
   ```

3. Check CloudWatch logs:
   ```bash
   aws logs describe-log-groups --region us-west-2
   ```

### ECR Repository doesn't exist
Create it first:
```bash
aws ecr create-repository \
  --repository-name startup-dose-api \
  --region us-west-2
```

### Docker login fails
Ensure you have AWS credentials configured:
```bash
aws sts get-caller-identity
```

## Next Steps

1. **Configure networking**: Set up security groups and subnets for your tasks
2. **Setup load balancer**: Add an ALB for traffic distribution
3. **Enable logging**: Configure CloudWatch log groups
4. **Setup CI/CD**: Automate deployments with GitHub Actions or similar
5. **Add monitoring**: Set up CloudWatch alarms and dashboards
