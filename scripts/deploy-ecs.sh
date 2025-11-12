#!/bin/bash

set -e

# Configuration
AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-296921411200}
AWS_REGION=${AWS_REGION:-us-west-2}
ECR_REPOSITORY_NAME="startup-dose-api"
ECS_CLUSTER_NAME=${ECS_CLUSTER_NAME:-unique-zebra-ow4l1t}
ECS_SERVICE_NAME=${ECS_SERVICE_NAME:-startup-dose-api-service}
ECS_TASK_FAMILY="startup-dose-api"
IMAGE_TAG=${IMAGE_TAG:-latest}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Validate inputs
if [ -z "$AWS_ACCOUNT_ID" ]; then
    print_warning "AWS_ACCOUNT_ID not set, using default: ${AWS_ACCOUNT_ID}"
fi

if [ -z "$ECS_CLUSTER_NAME" ]; then
    print_error "ECS_CLUSTER_NAME is not set"
    exit 1
fi

ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY_NAME}"

# Step 1: Build Docker image
print_info "Building Docker image..."
docker build -t ${ECR_REPOSITORY_NAME}:${IMAGE_TAG} .
docker tag ${ECR_REPOSITORY_NAME}:${IMAGE_TAG} ${ECR_URI}:${IMAGE_TAG}

# Step 2: Push to ECR
print_info "Logging in to ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

print_info "Pushing image to ECR..."
docker push ${ECR_URI}:${IMAGE_TAG}

# Step 3: Update ECS task definition
print_info "Updating ECS task definition..."

# Read the task definition template
TASK_DEF=$(cat ecs-task-definition.json | sed "s|YOUR_AWS_ACCOUNT_ID|${AWS_ACCOUNT_ID}|g" | sed "s|YOUR_AWS_REGION|${AWS_REGION}|g")

# Register the new task definition
NEW_TASK_DEF=$(aws ecs register-task-definition \
    --cli-input-json "$TASK_DEF" \
    --region ${AWS_REGION})

NEW_TASK_DEF_ARN=$(echo $NEW_TASK_DEF | grep -o "arn:aws:ecs:[^\"]*")

print_info "New task definition registered: $NEW_TASK_DEF_ARN"

# Step 4: Update ECS service
print_info "Updating ECS service..."
aws ecs update-service \
    --cluster ${ECS_CLUSTER_NAME} \
    --service ${ECS_SERVICE_NAME} \
    --task-definition ${ECS_TASK_FAMILY} \
    --force-new-deployment \
    --region ${AWS_REGION}

print_info "ECS service updated successfully!"
print_info "Deployment complete. Monitor your service at:"
print_info "https://console.aws.amazon.com/ecs/v2/clusters/${ECS_CLUSTER_NAME}/services/${ECS_SERVICE_NAME}"
