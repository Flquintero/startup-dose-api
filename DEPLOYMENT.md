# ECS Deployment Guide

This guide covers deploying the Startup Dose API to AWS ECS Fargate.

## Prerequisites

1. AWS Account with appropriate permissions
2. AWS CLI configured with credentials
3. Docker installed locally
4. ECR repository created in your AWS account

## Step 1: Create ECR Repository

```bash
aws ecr create-repository \
  --repository-name startup-dose-api \
  --region us-east-1
```

## Step 2: Setup Infrastructure with CloudFormation

### Create the VPC and Subnets (if needed)

You can use the default VPC or create a custom one. Note your VPC ID and Subnet IDs.

### Deploy the CloudFormation Stack

```bash
aws cloudformation create-stack \
  --stack-name startup-dose-api-stack \
  --template-body file://cloudformation/ecs-stack.yaml \
  --parameters \
    ParameterKey=EnvironmentName,ParameterValue=production \
    ParameterKey=VpcId,ParameterValue=vpc-xxxxxxxxx \
    ParameterKey=SubnetIds,ParameterValue="subnet-xxxxxxxxx,subnet-yyyyyyyyy" \
    ParameterKey=ContainerImage,ParameterValue=YOUR_AWS_ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/startup-dose-api:latest \
  --region us-east-1
```

Monitor the stack creation:
```bash
aws cloudformation describe-stack-events \
  --stack-name startup-dose-api-stack \
  --region us-east-1
```

## Step 3: Deploy Using Script

Set your environment variables:

```bash
export AWS_ACCOUNT_ID=123456789012
export AWS_REGION=us-east-1
export IMAGE_TAG=latest
```

Run the deployment script:

```bash
chmod +x scripts/deploy-ecs.sh
./scripts/deploy-ecs.sh
```

## Step 4: Manual Deployment (Alternative)

### Step 4a: Build and Push Docker Image

```bash
AWS_ACCOUNT_ID=123456789012
AWS_REGION=us-east-1

# Login to ECR
aws ecr get-login-password --region $AWS_REGION | \
  docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com

# Build and tag image
docker build -t startup-dose-api:latest .
docker tag startup-dose-api:latest $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/startup-dose-api:latest

# Push to ECR
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/startup-dose-api:latest
```

### Step 4b: Update Task Definition

```bash
# Register new task definition
aws ecs register-task-definition \
  --family startup-dose-api \
  --network-mode awsvpc \
  --requires-compatibilities FARGATE \
  --cpu 256 \
  --memory 512 \
  --container-definitions file://task-definition.json \
  --region $AWS_REGION
```

### Step 4c: Update ECS Service

```bash
aws ecs update-service \
  --cluster startup-dose-cluster-production \
  --service startup-dose-api-service-production \
  --task-definition startup-dose-api \
  --force-new-deployment \
  --region $AWS_REGION
```

## Monitoring

### View Service Status
```bash
aws ecs describe-services \
  --cluster startup-dose-cluster-production \
  --services startup-dose-api-service-production \
  --region $AWS_REGION
```

### View Logs
```bash
aws logs tail /ecs/startup-dose-api-production --follow --region $AWS_REGION
```

### View Task Events
```bash
aws ecs describe-tasks \
  --cluster startup-dose-cluster-production \
  --tasks <task-arn> \
  --region $AWS_REGION
```

## Scaling

### Manual Scaling
```bash
aws ecs update-service \
  --cluster startup-dose-cluster-production \
  --service startup-dose-api-service-production \
  --desired-count 4 \
  --region $AWS_REGION
```

### Auto Scaling (configured in CloudFormation)
The CloudFormation template automatically configures auto-scaling:
- Min tasks: 2
- Max tasks: 4
- CPU target: 70%
- Memory target: 80%

## Rollback

To rollback to a previous task definition:

```bash
# List previous task definitions
aws ecs list-task-definitions \
  --family-prefix startup-dose-api \
  --sort DESCENDING \
  --region $AWS_REGION

# Update service to use previous task definition
aws ecs update-service \
  --cluster startup-dose-cluster-production \
  --service startup-dose-api-service-production \
  --task-definition startup-dose-api:REVISION_NUMBER \
  --force-new-deployment \
  --region $AWS_REGION
```

## Health Checks

The API includes a health check endpoint at `/healthz` that returns:
```json
{
  "ok": true
}
```

This is automatically checked by:
- ECS health checks
- Load balancer health checks
- Can be monitored via CloudWatch

## Cleanup

To delete all resources:

```bash
# Delete CloudFormation stack
aws cloudformation delete-stack \
  --stack-name startup-dose-api-stack \
  --region $AWS_REGION

# Delete ECR repository (only if you want to remove the image)
aws ecr delete-repository \
  --repository-name startup-dose-api \
  --force \
  --region $AWS_REGION
```

## Environment Variables

To add environment variables to your tasks, update the task definition containerDefinitions section:

```json
"environment": [
  {
    "name": "LOG_LEVEL",
    "value": "info"
  },
  {
    "name": "PORT",
    "value": "8080"
  }
]
```

Or use Secrets Manager for sensitive values:

```json
"secrets": [
  {
    "name": "API_KEY",
    "valueFrom": "arn:aws:secretsmanager:region:account:secret:api-key"
  }
]
```

## Troubleshooting

### Task fails to start
- Check CloudWatch logs: `/ecs/startup-dose-api-production`
- Verify ECR image exists and is accessible
- Check IAM permissions for task execution role

### Health checks failing
- Verify API is running on port 8080
- Test `/healthz` endpoint manually
- Check network connectivity and security groups

### Service not reaching desired count
- Check task definition and resource limits
- Verify ECS cluster has capacity
- Review CloudWatch logs for errors

## References

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS Fargate Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/launch_types.html#launch-type-fargate)
- [CloudFormation ECS Resources](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/AWS_ECS.html)
