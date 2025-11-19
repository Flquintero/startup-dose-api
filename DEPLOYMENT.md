# ECS Deployment Guide

This guide covers deploying the Startup Dose API to AWS ECS Fargate with custom domain support (api.startupdose.com).

## Prerequisites

1. AWS Account with appropriate permissions
2. AWS CLI configured with credentials
3. Docker installed locally
4. ECR repository created in your AWS account
5. Access to GoDaddy DNS management for startupdose.com
6. VPC with at least 2 public subnets in different availability zones

## Quick Start (Using Makefile)

The easiest way to deploy is using the provided Makefile targets:

```bash
# 1. Request SSL certificate
make cert-request

# 2. Add DNS validation CNAME to GoDaddy (see certificate output)
# Wait for validation...

# 3. Check certificate status
make cert-check CERTIFICATE_ARN=arn:aws:acm:...

# 4. Build, push, and deploy
make deploy VPC_ID=vpc-xxx SUBNET_IDS=subnet-xxx,subnet-yyy CERTIFICATE_ARN=arn:aws:acm:...

# 5. Get DNS setup instructions
make dns-instructions

# 6. Add application CNAME to GoDaddy (points api.startupdose.com to ALB)
# Wait for DNS propagation...

# 7. Verify DNS is working
make dns-check

# 8. Test your API
curl https://api.startupdose.com/healthz
```

## Detailed Step-by-Step Guide

### Step 1: Request SSL Certificate

Request an SSL certificate from AWS Certificate Manager (ACM) for your custom domain:

```bash
make cert-request
```

Or manually:
```bash
./scripts/setup-certificate.sh --domain api.startupdose.com --region us-east-2
```

This will output:
- Certificate ARN (save this!)
- DNS validation CNAME record details

### Step 2: Add Certificate Validation CNAME to GoDaddy

**IMPORTANT**: This is the **first CNAME** you need to add (for certificate validation):

1. Log in to GoDaddy: https://dcc.godaddy.com/domains
2. Select your domain: startupdose.com
3. Go to DNS settings
4. Add the CNAME record from the certificate request output
   - Example: `_abc123.api.startupdose.com` → `_xyz456.acm-validations.aws.`
5. Save the record

Wait 5-30 minutes for validation to complete.

### Step 3: Verify Certificate is Validated

Check the certificate status:

```bash
make cert-check CERTIFICATE_ARN=arn:aws:acm:us-east-2:123456789:certificate/xxx
```

Or manually:
```bash
./scripts/setup-certificate.sh --certificate-arn arn:aws:acm:... --region us-east-2
```

Wait until status shows `ISSUED` before proceeding.

### Step 4: Create ECR Repository (if not exists)

```bash
aws ecr create-repository \
  --repository-name startup-dose/api \
  --region us-east-2
```

### Step 5: Deploy with CloudFormation

Build Docker image, push to ECR, and deploy the CloudFormation stack:

```bash
make deploy \
  VPC_ID=vpc-xxx \
  SUBNET_IDS=subnet-xxx,subnet-yyy \
  CERTIFICATE_ARN=arn:aws:acm:us-east-2:123456789:certificate/xxx \
  TAG=v1.0.0
```

This will:
1. Login to ECR
2. Build Docker image
3. Tag and push to ECR
4. Create/update CloudFormation stack with:
   - Application Load Balancer (ALB)
   - ECS Fargate cluster and service
   - Auto-scaling configuration
   - HTTPS listener with your SSL certificate
   - HTTP → HTTPS redirect

Or deploy manually:
```bash
./scripts/deploy-cloudformation.sh \
  --vpc-id vpc-xxx \
  --subnet-ids subnet-xxx,subnet-yyy \
  --certificate-arn arn:aws:acm:... \
  --stack-name startup-dose-api-production \
  --image-tag v1.0.0
```

### Step 6: Add Application CNAME to GoDaddy

**IMPORTANT**: This is the **second CNAME** you need to add (routes traffic to your API):

After deployment completes, get the Load Balancer DNS:

```bash
make dns-instructions
```

This will show you exactly what CNAME to add:
1. Log in to GoDaddy: https://dcc.godaddy.com/domains
2. Select your domain: startupdose.com
3. Go to DNS settings
4. Add a CNAME record:
   - **Type**: CNAME
   - **Name**: api
   - **Value**: startup-dose-api-alb-production-123456789.us-east-2.elb.amazonaws.com
   - **TTL**: 600 (10 minutes)
5. Save the record

### Step 7: Verify DNS Configuration

Wait 5-30 minutes for DNS propagation, then verify:

```bash
make dns-check
```

This will check if `api.startupdose.com` correctly points to your Load Balancer.

### Step 8: Test Your API

```bash
curl https://api.startupdose.com/healthz
```

Expected response:
```json
{
  "ok": true
}
```

## Understanding the Two CNAME Records

Your deployment requires **two different CNAME records**:

### 1. Certificate Validation CNAME (Temporary, but keep it)
- **Purpose**: Proves you own the domain to AWS
- **Example**: `_abc123def.api.startupdose.com` → `_xyz789.acm-validations.aws.`
- **When**: Added during `make cert-request` (Step 2)
- **Note**: AWS periodically re-validates, so keep this record

### 2. Application CNAME (Permanent)
- **Purpose**: Routes traffic to your API
- **Example**: `api.startupdose.com` → `startup-dose-api-alb-xxx.us-east-2.elb.amazonaws.com`
- **When**: Added after deployment (Step 6)
- **Note**: This is how users access your API

## Available Makefile Targets

```bash
# Development
make run          # Run the server locally
make build-local  # Build the Go binary
make test         # Run tests

# Docker & ECR
make login        # Login to ECR
make docker-build # Build Docker image
make tag          # Tag image for ECR (TAG=v0.1.0)
make push         # Push image to ECR
make all          # Login, build, tag, and push
make release      # Alias for 'make all'

# Certificate Management
make cert-request                    # Request SSL certificate
make cert-check CERTIFICATE_ARN=...  # Check certificate status
make cert-list                       # List all certificates

# Deployment
make deploy-stack VPC_ID=... SUBNET_IDS=... CERTIFICATE_ARN=... # Deploy CloudFormation stack
make deploy VPC_ID=... SUBNET_IDS=... CERTIFICATE_ARN=...       # Full deployment (build + deploy)

# DNS Setup
make dns-instructions  # Show GoDaddy DNS setup instructions
make dns-check         # Check if DNS is configured correctly

# Help
make help  # Show all available targets
```

## Using Scripts Directly (Without Makefile)

If you prefer not to use the Makefile:

### 1. Request Certificate
```bash
./scripts/setup-certificate.sh --domain api.startupdose.com --region us-east-2
```

### 2. Check Certificate Status
```bash
./scripts/setup-certificate.sh --certificate-arn arn:aws:acm:... --region us-east-2
```

### 3. Build and Push Docker Image
```bash
# Set variables
export ECR_REGISTRY="799771184733.dkr.ecr.us-east-2.amazonaws.com"
export ECR_REPO="startup-dose/api"
export TAG="v1.0.0"
export REGION="us-east-2"

# Login to ECR
aws ecr get-login-password --region $REGION | \
  docker login --username AWS --password-stdin $ECR_REGISTRY

# Build, tag, and push
docker build -t startup-dose-api:$TAG .
docker tag startup-dose-api:$TAG $ECR_REGISTRY/$ECR_REPO:$TAG
docker push $ECR_REGISTRY/$ECR_REPO:$TAG
```

### 4. Deploy CloudFormation Stack
```bash
./scripts/deploy-cloudformation.sh \
  --vpc-id vpc-xxx \
  --subnet-ids subnet-xxx,subnet-yyy \
  --certificate-arn arn:aws:acm:us-east-2:123456789:certificate/xxx \
  --stack-name startup-dose-api-production \
  --image-tag v1.0.0
```

### 5. Get DNS Instructions
```bash
./scripts/setup-dns.sh
```

### 6. Check DNS Status
```bash
./scripts/setup-dns.sh --check
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

## Manual ALB Setup (Alternative to CloudFormation)

If CloudFormation deployment times out or fails, you can manually set up the Application Load Balancer:

### Prerequisites
1. Ensure Dockerfile includes `wget` for health checks:
```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
```

2. Build and push Docker image to ECR:
```bash
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin YOUR_ACCOUNT_ID.dkr.ecr.us-east-2.amazonaws.com
docker build -t startup-dose-api .
docker tag startup-dose-api:latest YOUR_ACCOUNT_ID.dkr.ecr.us-east-2.amazonaws.com/startup-dose/api:latest
docker push YOUR_ACCOUNT_ID.dkr.ecr.us-east-2.amazonaws.com/startup-dose/api:latest
```

### Step 1: Delete Failed CloudFormation Stack (if exists)
```bash
aws cloudformation delete-stack --stack-name startup-dose-api-stack --region us-east-2
```

### Step 2: Create Security Groups

Create ALB security group:
```bash
aws ec2 create-security-group \
  --group-name startup-dose-alb-sg \
  --description "Security group for Startup Dose ALB" \
  --vpc-id vpc-YOUR_VPC_ID \
  --region us-east-2
```

Add inbound rules for ALB (HTTP and HTTPS):
```bash
# Get the security group ID from previous command output
ALB_SG_ID="sg-xxxxx"

# Allow HTTPS from anywhere
aws ec2 authorize-security-group-ingress \
  --group-id $ALB_SG_ID \
  --protocol tcp \
  --port 443 \
  --cidr 0.0.0.0/0 \
  --region us-east-2

# Allow HTTP from anywhere
aws ec2 authorize-security-group-ingress \
  --group-id $ALB_SG_ID \
  --protocol tcp \
  --port 80 \
  --cidr 0.0.0.0/0 \
  --region us-east-2
```

Update ECS security group to allow traffic from ALB:
```bash
ECS_SG_ID="sg-yyyyy"  # Your existing ECS security group

aws ec2 authorize-security-group-ingress \
  --group-id $ECS_SG_ID \
  --protocol tcp \
  --port 8080 \
  --source-group $ALB_SG_ID \
  --region us-east-2
```

### Step 3: Create Application Load Balancer
```bash
aws elbv2 create-load-balancer \
  --name startup-dose-api-alb \
  --subnets subnet-xxxxx subnet-yyyyy \
  --security-groups $ALB_SG_ID \
  --scheme internet-facing \
  --type application \
  --ip-address-type ipv4 \
  --region us-east-2
```

Save the ALB ARN from the output.

### Step 4: Create Target Group
```bash
aws elbv2 create-target-group \
  --name startup-dose-api-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-YOUR_VPC_ID \
  --target-type ip \
  --health-check-enabled \
  --health-check-protocol HTTP \
  --health-check-path /healthz \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 10 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3 \
  --region us-east-2
```

Save the Target Group ARN from the output.

### Step 5: Create HTTPS Listener
```bash
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-2:YOUR_ACCOUNT:loadbalancer/app/startup-dose-api-alb/xxxxx \
  --protocol HTTPS \
  --port 443 \
  --certificates CertificateArn=arn:aws:acm:us-east-2:YOUR_ACCOUNT:certificate/xxxxx \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-2:YOUR_ACCOUNT:targetgroup/startup-dose-api-tg/xxxxx \
  --region us-east-2
```

### Step 6: Create HTTP Listener (Redirect to HTTPS)
```bash
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-2:YOUR_ACCOUNT:loadbalancer/app/startup-dose-api-alb/xxxxx \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=redirect,RedirectConfig='{Protocol=HTTPS,Port=443,StatusCode=HTTP_301}' \
  --region us-east-2
```

### Step 7: Create Task Definition with Environment Variables

Create a JSON file with your task definition (remember to use environment variables, not hardcoded secrets):
```bash
cat > task-def.json << 'EOF'
{
  "family": "startup-dose-api-fargate",
  "executionRoleArn": "arn:aws:iam::YOUR_ACCOUNT:role/ecsTaskExecutionRole",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "startup-dose-api",
      "image": "YOUR_ACCOUNT.dkr.ecr.us-east-2.amazonaws.com/startup-dose/api:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {"name": "PORT", "value": "8080"},
        {"name": "LOG_LEVEL", "value": "info"},
        {"name": "AWS_REGION", "value": "us-east-2"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/startup-dose-api",
          "awslogs-region": "us-east-2",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "wget --quiet --tries=1 --spider http://localhost:8080/healthz || exit 1"
        ],
        "interval": 30,
        "timeout": 10,
        "retries": 3,
        "startPeriod": 10
      }
    }
  ]
}
EOF
```

Register the task definition:
```bash
aws ecs register-task-definition --cli-input-json file://task-def.json --region us-east-2
```

### Step 8: Create ECS Service with Load Balancer
```bash
aws ecs create-service \
  --cluster startup-dose-cluster \
  --service-name startup-dose-api-alb \
  --task-definition startup-dose-api-fargate:REVISION \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxxx,subnet-yyyyy],securityGroups=[$ECS_SG_ID],assignPublicIp=ENABLED}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:us-east-2:YOUR_ACCOUNT:targetgroup/startup-dose-api-tg/xxxxx,containerName=startup-dose-api,containerPort=8080" \
  --health-check-grace-period-seconds 60 \
  --region us-east-2
```

### Step 9: Verify Target Health
```bash
aws elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-2:YOUR_ACCOUNT:targetgroup/startup-dose-api-tg/xxxxx \
  --region us-east-2
```

Wait until targets show `State: healthy`.

### Step 10: Update DNS CNAME
Get the ALB DNS name:
```bash
aws elbv2 describe-load-balancers \
  --names startup-dose-api-alb \
  --query 'LoadBalancers[0].DNSName' \
  --output text \
  --region us-east-2
```

Update your GoDaddy CNAME record:
- **Name**: api
- **Value**: [ALB DNS name from above]
- **TTL**: 600

### Step 11: Test the Deployment
```bash
# Wait for DNS propagation (5-30 minutes)
dig api.startupdose.com CNAME +short

# Test the API
curl https://api.startupdose.com/healthz
```

## Updating Environment Variables

If you need to update environment variables after deployment:

1. Create new task definition with updated environment variables
2. Register the new task definition
3. Update the ECS service to use the new task definition:

```bash
aws ecs update-service \
  --cluster startup-dose-cluster \
  --service startup-dose-api-alb \
  --task-definition startup-dose-api-fargate:NEW_REVISION \
  --force-new-deployment \
  --region us-east-2
```

4. Wait for deployment to complete:
```bash
aws ecs describe-services \
  --cluster startup-dose-cluster \
  --services startup-dose-api-alb \
  --region us-east-2
```

## Troubleshooting

### CloudFormation deployment times out
- Issue: CloudFormation may timeout waiting for ECS service to stabilize
- Solution: Use the manual ALB setup process described above
- Note: Ensure Dockerfile includes `wget` package for health checks

### Task fails to start
- Check CloudWatch logs: `/ecs/startup-dose-api`
- Verify ECR image exists and is accessible
- Check IAM permissions for task execution role
- Ensure task definition has correct executionRoleArn

### Health checks failing
- Verify API is running on port 8080
- Test `/healthz` endpoint manually inside container
- Check that Dockerfile includes `wget`: `RUN apk --no-cache add ca-certificates wget`
- Check network connectivity and security groups
- Review health check command in task definition

### Service not reaching desired count
- Check task definition and resource limits
- Verify ECS cluster has capacity
- Review CloudWatch logs for errors
- Check if tasks are failing health checks

### 502 Bad Gateway errors
- Check if all tasks have environment variables configured
- Verify targets are healthy in target group
- Look for tasks running old task definitions without proper configuration
- Force new deployment to replace unhealthy tasks

### Environment variables not loaded
- Ensure task definition includes all required environment variables
- Check task execution role has permissions to access secrets (if using Secrets Manager)
- Verify environment variables are in the correct format in task definition
- Force new deployment after updating task definition

## References

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS Fargate Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/launch_types.html#launch-type-fargate)
- [CloudFormation ECS Resources](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/AWS_ECS.html)
