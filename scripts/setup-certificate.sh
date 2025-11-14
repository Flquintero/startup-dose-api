#!/bin/bash

set -e

# Configuration
AWS_REGION=${AWS_REGION:-us-east-2}
DOMAIN_NAME=${DOMAIN_NAME:-api.startupdose.com}

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

# Parse command line arguments
ACTION="request"
CERTIFICATE_ARN=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN_NAME="$2"
            shift 2
            ;;
        --region)
            AWS_REGION="$2"
            shift 2
            ;;
        --certificate-arn)
            CERTIFICATE_ARN="$2"
            ACTION="check"
            shift 2
            ;;
        --list)
            ACTION="list"
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --domain DOMAIN          Domain name (default: api.startupdose.com)"
            echo "  --region REGION          AWS region (default: us-east-2)"
            echo "  --certificate-arn ARN    Check status of existing certificate"
            echo "  --list                   List all certificates in the region"
            echo "  --help                   Show this help message"
            echo ""
            echo "Example:"
            echo "  # Request a new certificate"
            echo "  $0 --domain api.startupdose.com"
            echo ""
            echo "  # Check certificate status"
            echo "  $0 --certificate-arn arn:aws:acm:us-east-2:123456789:certificate/xxx"
            echo ""
            echo "  # List all certificates"
            echo "  $0 --list"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

print_header "SSL Certificate Setup for $DOMAIN_NAME"

case $ACTION in
    request)
        print_info "Requesting SSL certificate for domain: $DOMAIN_NAME"
        print_info "Region: $AWS_REGION"
        echo ""

        RESULT=$(aws acm request-certificate \
            --domain-name "$DOMAIN_NAME" \
            --validation-method DNS \
            --region "$AWS_REGION" \
            --output json)

        CERTIFICATE_ARN=$(echo "$RESULT" | grep -o '"CertificateArn": "[^"]*"' | cut -d'"' -f4)

        print_info "Certificate requested successfully!"
        echo ""
        echo "Certificate ARN: $CERTIFICATE_ARN"
        echo ""

        print_warning "Certificate is pending validation. You need to add DNS records."
        echo ""

        sleep 3
        print_info "Fetching validation records..."
        echo ""

        CERT_DETAILS=$(aws acm describe-certificate \
            --certificate-arn "$CERTIFICATE_ARN" \
            --region "$AWS_REGION" \
            --output json)

        echo "$CERT_DETAILS" | grep -A 5 "DomainValidationOptions" || true

        print_header "Next Steps"
        echo "1. Add the DNS validation record to your DNS provider"
        echo ""
        echo "To see validation details, run:"
        echo "  aws acm describe-certificate --certificate-arn $CERTIFICATE_ARN --region $AWS_REGION"
        echo ""
        echo "2. Wait for validation (usually 5-30 minutes)"
        echo ""
        echo "3. Check validation status with:"
        echo "  $0 --certificate-arn $CERTIFICATE_ARN"
        echo ""
        echo "4. Once validated, deploy with:"
        echo "  ./scripts/deploy-cloudformation.sh --certificate-arn $CERTIFICATE_ARN --vpc-id <vpc-id> --subnet-ids <subnet-ids>"
        ;;

    check)
        if [ -z "$CERTIFICATE_ARN" ]; then
            print_error "Certificate ARN is required for check action"
            exit 1
        fi

        print_info "Checking certificate status..."
        echo ""

        CERT_DETAILS=$(aws acm describe-certificate \
            --certificate-arn "$CERTIFICATE_ARN" \
            --region "$AWS_REGION" \
            --output json)

        STATUS=$(echo "$CERT_DETAILS" | grep -o '"Status": "[^"]*"' | head -1 | cut -d'"' -f4)
        DOMAIN=$(echo "$CERT_DETAILS" | grep -o '"DomainName": "[^"]*"' | head -1 | cut -d'"' -f4)

        echo "Domain:     $DOMAIN"
        echo "Status:     $STATUS"
        echo "ARN:        $CERTIFICATE_ARN"
        echo ""

        case $STATUS in
            ISSUED)
                print_info "✓ Certificate is validated and ready to use!"
                echo ""
                print_info "Deploy with this certificate using:"
                echo "  ./scripts/deploy-cloudformation.sh --certificate-arn $CERTIFICATE_ARN --vpc-id <vpc-id> --subnet-ids <subnet-ids>"
                ;;
            PENDING_VALIDATION)
                print_warning "Certificate is pending validation"
                echo ""
                print_info "DNS Validation Records:"
                echo "$CERT_DETAILS" | grep -A 10 "ResourceRecord" || true
                echo ""
                print_info "Add the CNAME record above to your DNS provider"
                ;;
            VALIDATION_TIMED_OUT|FAILED)
                print_error "Certificate validation failed or timed out"
                print_info "You may need to request a new certificate"
                ;;
            *)
                print_warning "Certificate status: $STATUS"
                ;;
        esac
        ;;

    list)
        print_info "Listing all certificates in region: $AWS_REGION"
        echo ""

        aws acm list-certificates \
            --region "$AWS_REGION" \
            --output table \
            --query 'CertificateSummaryList[].[DomainName,Status,CertificateArn]'
        ;;
esac
