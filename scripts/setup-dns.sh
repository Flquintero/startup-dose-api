#!/bin/bash

set -e

# Configuration
STACK_NAME=${STACK_NAME:-startup-dose-api-production}
AWS_REGION=${AWS_REGION:-us-east-2}
DOMAIN_NAME=${DOMAIN_NAME:-api.startupdose.com}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

print_step() {
    echo -e "${CYAN}➜${NC} $1"
}

# Parse command line arguments
ACTION="instructions"
CHECK_DNS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --stack-name)
            STACK_NAME="$2"
            shift 2
            ;;
        --domain)
            DOMAIN_NAME="$2"
            shift 2
            ;;
        --region)
            AWS_REGION="$2"
            shift 2
            ;;
        --check)
            CHECK_DNS=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --stack-name NAME    CloudFormation stack name (default: startup-dose-api-production)"
            echo "  --domain DOMAIN      Domain name (default: api.startupdose.com)"
            echo "  --region REGION      AWS region (default: us-east-2)"
            echo "  --check              Check if DNS is configured correctly"
            echo "  --help               Show this help message"
            echo ""
            echo "Examples:"
            echo "  # Show DNS setup instructions"
            echo "  $0"
            echo ""
            echo "  # Check DNS configuration"
            echo "  $0 --check"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

print_header "GoDaddy DNS Setup for $DOMAIN_NAME"

# Get Load Balancer DNS from CloudFormation stack
print_info "Fetching Load Balancer DNS from CloudFormation stack..."
echo ""

LB_DNS=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$AWS_REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`LoadBalancerDNS`].OutputValue' \
    --output text 2>/dev/null || echo "")

if [ -z "$LB_DNS" ]; then
    print_error "Could not retrieve Load Balancer DNS from stack: $STACK_NAME"
    print_info "Make sure the CloudFormation stack is deployed successfully"
    print_info "Run: aws cloudformation describe-stacks --stack-name $STACK_NAME --region $AWS_REGION"
    exit 1
fi

print_info "Load Balancer DNS: $LB_DNS"
echo ""

if [ "$CHECK_DNS" = true ]; then
    print_header "Checking DNS Configuration"
    echo ""

    print_info "Current DNS resolution for $DOMAIN_NAME:"
    CURRENT_DNS=$(dig +short "$DOMAIN_NAME" | tail -n1 || echo "NOT_CONFIGURED")

    if [ "$CURRENT_DNS" = "NOT_CONFIGURED" ] || [ -z "$CURRENT_DNS" ]; then
        print_error "DNS is not configured or not propagated yet"
        echo ""
        print_warning "Please add the CNAME record to GoDaddy and wait for DNS propagation"
    else
        echo "  Resolves to: $CURRENT_DNS"
        echo ""

        # Check if it resolves to the Load Balancer
        LB_IPS=$(dig +short "$LB_DNS" | sort)
        DOMAIN_IPS=$(dig +short "$DOMAIN_NAME" | sort)

        if [ "$LB_IPS" = "$DOMAIN_IPS" ]; then
            print_info "✓ DNS is correctly configured!"
            print_info "$DOMAIN_NAME points to the Load Balancer"
            echo ""
            print_info "Test your API:"
            echo "  curl https://$DOMAIN_NAME/healthz"
        else
            print_warning "DNS is configured but may not be pointing to the correct Load Balancer"
            echo ""
            echo "Expected IPs (Load Balancer):"
            echo "$LB_IPS" | sed 's/^/  /'
            echo ""
            echo "Actual IPs ($DOMAIN_NAME):"
            echo "$DOMAIN_IPS" | sed 's/^/  /'
            echo ""
            print_warning "Please verify the CNAME record in GoDaddy"
        fi
    fi
else
    print_header "GoDaddy DNS Setup Instructions"
    echo ""

    print_info "You need to add a CNAME record in GoDaddy to point your domain to the Load Balancer"
    echo ""

    print_header "Step-by-Step Instructions"
    echo ""

    print_step "1. Log in to your GoDaddy account"
    echo "   https://dcc.godaddy.com/domains"
    echo ""

    print_step "2. Select your domain: startupdose.com"
    echo ""

    print_step "3. Click on 'DNS' or 'Manage DNS'"
    echo ""

    print_step "4. Add a new CNAME record with these values:"
    echo ""
    echo "   ┌─────────────────────────────────────────────────┐"
    echo "   │ Type:  CNAME                                    │"
    echo "   │ Name:  api                                      │"
    echo "   │ Value: $LB_DNS"
    echo "   │ TTL:   600 (or 10 minutes)                      │"
    echo "   └─────────────────────────────────────────────────┘"
    echo ""

    print_step "5. Save the record"
    echo ""

    print_step "6. Wait for DNS propagation (5-30 minutes)"
    echo ""

    print_header "After DNS Setup"
    echo ""

    print_info "To check if DNS is configured correctly, run:"
    echo "  $0 --check"
    echo ""

    print_info "To test your API once DNS is ready:"
    echo "  curl https://$DOMAIN_NAME/healthz"
    echo ""

    print_warning "Note: This is the APPLICATION CNAME, different from the certificate validation CNAME"
    print_info "The certificate validation CNAME should already be set up and validated"
fi

print_header "Summary"
echo ""
echo "Stack:        $STACK_NAME"
echo "Region:       $AWS_REGION"
echo "Domain:       $DOMAIN_NAME"
echo "Load Balancer: $LB_DNS"
echo ""
