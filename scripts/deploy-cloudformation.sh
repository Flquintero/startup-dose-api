#!/bin/bash

set -e

# Configuration
STACK_NAME=${STACK_NAME:-startup-dose-api-production}
AWS_REGION=${AWS_REGION:-us-east-2}
ENVIRONMENT_NAME=${ENVIRONMENT_NAME:-production}
CONTAINER_PORT=${CONTAINER_PORT:-8080}
DOMAIN_NAME=${DOMAIN_NAME:-api.startupdose.com}

# ECR Configuration (must match Makefile)
ECR_REGISTRY="799771184733.dkr.ecr.us-east-2.amazonaws.com"
ECR_REPO="startup-dose/api"
IMAGE_TAG=${IMAGE_TAG:-latest}
CONTAINER_IMAGE="${ECR_REGISTRY}/${ECR_REPO}:${IMAGE_TAG}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Function to check if stack exists
stack_exists() {
    aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        &>/dev/null
}

# Function to get stack status
get_stack_status() {
    aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        --query 'Stacks[0].StackStatus' \
        --output text 2>/dev/null || echo "DOES_NOT_EXIST"
}

# Function to wait for stack operation
wait_for_stack() {
    local operation=$1
    print_info "Waiting for stack $operation to complete..."

    aws cloudformation wait "stack-${operation}-complete" \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" || {
        print_error "Stack $operation failed!"

        # Show stack events
        print_info "Recent stack events:"
        aws cloudformation describe-stack-events \
            --stack-name "$STACK_NAME" \
            --region "$AWS_REGION" \
            --max-items 10 \
            --query 'StackEvents[?ResourceStatus==`CREATE_FAILED` || ResourceStatus==`UPDATE_FAILED`].[Timestamp,ResourceType,LogicalResourceId,ResourceStatusReason]' \
            --output table

        exit 1
    }
}

# Function to display stack outputs
display_outputs() {
    print_header "Stack Outputs"
    aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        --query 'Stacks[0].Outputs[].[OutputKey,OutputValue,Description]' \
        --output table
}

# Parse command line arguments
CERTIFICATE_ARN=""
VPC_ID=""
SUBNET_IDS=""
SKIP_CERTIFICATE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --certificate-arn)
            CERTIFICATE_ARN="$2"
            shift 2
            ;;
        --vpc-id)
            VPC_ID="$2"
            shift 2
            ;;
        --subnet-ids)
            SUBNET_IDS="$2"
            shift 2
            ;;
        --skip-certificate)
            SKIP_CERTIFICATE=true
            shift
            ;;
        --stack-name)
            STACK_NAME="$2"
            shift 2
            ;;
        --image-tag)
            IMAGE_TAG="$2"
            CONTAINER_IMAGE="${ECR_REGISTRY}/${ECR_REPO}:${IMAGE_TAG}"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --certificate-arn ARN    ACM certificate ARN for HTTPS"
            echo "  --vpc-id VPC_ID          VPC ID for deployment"
            echo "  --subnet-ids IDS         Comma-separated subnet IDs (at least 2)"
            echo "  --skip-certificate       Deploy without HTTPS/custom domain"
            echo "  --stack-name NAME        CloudFormation stack name (default: startup-dose-api-production)"
            echo "  --image-tag TAG          Docker image tag (default: latest)"
            echo "  --help                   Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  AWS_REGION              AWS region (default: us-east-2)"
            echo "  ENVIRONMENT_NAME        Environment name (default: production)"
            echo "  DOMAIN_NAME             Custom domain (default: api.startupdose.com)"
            echo ""
            echo "Example:"
            echo "  $0 --vpc-id vpc-123 --subnet-ids subnet-123,subnet-456 --certificate-arn arn:aws:acm:..."
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

print_header "CloudFormation Deployment Script"

# Validate required parameters
if [ -z "$VPC_ID" ]; then
    print_error "VPC ID is required. Use --vpc-id option or set VPC_ID environment variable"
    print_info "To find your VPCs, run: aws ec2 describe-vpcs --region $AWS_REGION"
    exit 1
fi

if [ -z "$SUBNET_IDS" ]; then
    print_error "Subnet IDs are required. Use --subnet-ids option or set SUBNET_IDS environment variable"
    print_info "To find subnets in your VPC, run: aws ec2 describe-subnets --filters Name=vpc-id,Values=$VPC_ID --region $AWS_REGION"
    exit 1
fi

# Build parameters array
PARAMETERS=(
    "ParameterKey=EnvironmentName,ParameterValue=$ENVIRONMENT_NAME"
    "ParameterKey=VpcId,ParameterValue=$VPC_ID"
    "ParameterKey=SubnetIds,ParameterValue=\"$SUBNET_IDS\""
    "ParameterKey=ContainerImage,ParameterValue=$CONTAINER_IMAGE"
    "ParameterKey=ContainerPort,ParameterValue=$CONTAINER_PORT"
)

# Add domain parameters if certificate is provided
if [ "$SKIP_CERTIFICATE" = false ]; then
    if [ -z "$CERTIFICATE_ARN" ]; then
        print_warning "No certificate ARN provided. Deploying without HTTPS."
        print_info "To enable HTTPS, request a certificate and provide --certificate-arn"
        print_info "To skip this warning, use --skip-certificate flag"
        PARAMETERS+=(
            "ParameterKey=DomainName,ParameterValue="
            "ParameterKey=CertificateArn,ParameterValue="
        )
    else
        print_info "Deploying with HTTPS enabled for domain: $DOMAIN_NAME"
        PARAMETERS+=(
            "ParameterKey=DomainName,ParameterValue=$DOMAIN_NAME"
            "ParameterKey=CertificateArn,ParameterValue=$CERTIFICATE_ARN"
        )
    fi
else
    PARAMETERS+=(
        "ParameterKey=DomainName,ParameterValue="
        "ParameterKey=CertificateArn,ParameterValue="
    )
fi

# Display configuration
print_info "Configuration:"
echo "  Stack Name:       $STACK_NAME"
echo "  Region:           $AWS_REGION"
echo "  Environment:      $ENVIRONMENT_NAME"
echo "  Container Image:  $CONTAINER_IMAGE"
echo "  VPC ID:           $VPC_ID"
echo "  Subnet IDs:       $SUBNET_IDS"
if [ "$SKIP_CERTIFICATE" = false ] && [ -n "$CERTIFICATE_ARN" ]; then
    echo "  Domain:           $DOMAIN_NAME"
    echo "  Certificate ARN:  ${CERTIFICATE_ARN:0:50}..."
fi
echo ""

# Check if stack exists
STACK_STATUS=$(get_stack_status)
print_info "Stack status: $STACK_STATUS"

if [ "$STACK_STATUS" = "DOES_NOT_EXIST" ]; then
    print_info "Creating new CloudFormation stack..."

    aws cloudformation create-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://cloudformation/ecs-stack.yaml \
        --parameters "${PARAMETERS[@]}" \
        --capabilities CAPABILITY_IAM \
        --region "$AWS_REGION" \
        --tags Key=Environment,Value="$ENVIRONMENT_NAME" Key=Project,Value=startup-dose-api

    wait_for_stack "create"
    print_info "Stack created successfully!"
else
    print_info "Updating existing CloudFormation stack..."

    aws cloudformation update-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://cloudformation/ecs-stack.yaml \
        --parameters "${PARAMETERS[@]}" \
        --capabilities CAPABILITY_IAM \
        --region "$AWS_REGION" || {

        # Check if there are no updates
        if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$AWS_REGION" 2>&1 | grep -q "No updates are to be performed"; then
            print_warning "No updates to be performed on stack"
        else
            print_error "Stack update failed!"
            exit 1
        fi
    }

    # Only wait if update was initiated
    CURRENT_STATUS=$(get_stack_status)
    if [[ "$CURRENT_STATUS" == *"IN_PROGRESS"* ]]; then
        wait_for_stack "update"
        print_info "Stack updated successfully!"
    fi
fi

# Display outputs
display_outputs

print_header "Deployment Complete!"

# Get Load Balancer DNS for DNS setup reminder
LB_DNS=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$AWS_REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`LoadBalancerDNS`].OutputValue' \
    --output text 2>/dev/null || echo "")

if [ -n "$LB_DNS" ] && [ -n "$CERTIFICATE_ARN" ]; then
    print_info "Next steps:"
    echo "  1. Create a DNS record for $DOMAIN_NAME"
    echo "     Type: CNAME (or A-Alias for Route 53)"
    echo "     Value: $LB_DNS"
    echo ""
    echo "  2. Wait for DNS propagation (5-30 minutes)"
    echo ""
    echo "  3. Test your API:"
    echo "     curl https://$DOMAIN_NAME/healthz"
fi

print_info "Monitor your deployment at:"
echo "  https://console.aws.amazon.com/cloudformation/home?region=${AWS_REGION}#/stacks?filteringText=${STACK_NAME}"
