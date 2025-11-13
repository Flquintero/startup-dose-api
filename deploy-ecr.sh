#!/bin/bash

set -euo pipefail

# Configuration
ECR_REGISTRY="799771184733.dkr.ecr.us-east-2.amazonaws.com"
ECR_REPO="startup-dose/api"
IMAGE_NAME="startup-dose-api"
REGION="us-east-2"
TAG="${1:-latest}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== ECR Deploy Script ===${NC}"
echo "Tag: $TAG"
echo "ECR URI: $ECR_REGISTRY/$ECR_REPO:$TAG"
echo ""

# Step 1: Login to ECR
echo -e "${GREEN}[1/4]${NC} Logging into ECR..."
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ECR_REGISTRY

# Step 2: Build Docker image
echo -e "${GREEN}[2/4]${NC} Building Docker image..."
docker build -t $IMAGE_NAME:$TAG .

# Step 3: Tag the image
echo -e "${GREEN}[3/4]${NC} Tagging image as $ECR_REGISTRY/$ECR_REPO:$TAG..."
docker tag $IMAGE_NAME:$TAG $ECR_REGISTRY/$ECR_REPO:$TAG

# Step 4: Push to ECR
echo -e "${GREEN}[4/4]${NC} Pushing image to ECR..."
docker push $ECR_REGISTRY/$ECR_REPO:$TAG

echo ""
echo -e "${GREEN}✓ Successfully deployed to ECR!${NC}"
echo "Image: $ECR_REGISTRY/$ECR_REPO:$TAG"
