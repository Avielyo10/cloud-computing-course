#!/bin/bash

set -e

# API details from Terraform deployment
API_URL="1zes1gobgf.execute-api.il-central-1.amazonaws.com/prod"
DYNAMO_TABLE="parkingTickets"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE}Testing Parking Lot API Deployment${NC}"
echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE}API URL: ${NC}https://$API_URL"
echo -e "${BLUE}DynamoDB Table: ${NC}$DYNAMO_TABLE"

# Test entry endpoint
echo -e "\n${BLUE}Testing entry endpoint...${NC}"
PLATE="TEST-$(date +%s)"
PARKING_LOT=1

echo -e "Sending request: POST /entry?plate=$PLATE&parkingLot=$PARKING_LOT"
ENTRY_RESPONSE=$(curl -s -X POST "https://$API_URL/entry?plate=$PLATE&parkingLot=$PARKING_LOT")
echo "Response: $ENTRY_RESPONSE"

# Extract ticket ID from the response
TICKET_ID=$(echo $ENTRY_RESPONSE | grep -o '"ticketId":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TICKET_ID" ]; then
    echo -e "${RED}Failed to extract ticket ID from response${NC}"
    exit 1
else
    echo -e "${GREEN}Successfully created ticket: $TICKET_ID${NC}"
fi

aws dynamodb get-item --table-name $DYNAMO_TABLE --key "{\"ticketId\":{\"S\":\"$TICKET_ID\"}}"

# Wait for a few seconds to simulate parking time
echo -e "\n${BLUE}Simulating parking time (10 seconds)...${NC}"
sleep 10

# Test exit endpoint
echo -e "\n${BLUE}Testing exit endpoint...${NC}"
echo -e "Sending request: POST /exit?ticketId=$TICKET_ID"
EXIT_RESPONSE=$(curl -s -X POST "https://$API_URL/exit?ticketId=$TICKET_ID")
echo "Response: $EXIT_RESPONSE"

# Check if response contains expected fields
if echo $EXIT_RESPONSE | grep -q "charge"; then
    echo -e "${GREEN}Successfully processed exit${NC}"
else
    echo -e "${RED}Failed to process exit${NC}"
    exit 1
fi

# Use AWS CLI to verify the ticket is in DynamoDB with status = out
if command -v aws &> /dev/null; then
    echo -e "\n${BLUE}Verifying DynamoDB record...${NC}"
    echo -e "aws dynamodb get-item --table-name $DYNAMO_TABLE --key '{\"ticketId\":{\"S\":\"$TICKET_ID\"}}'"
    
    AWS_RECORD=$(aws dynamodb get-item --table-name $DYNAMO_TABLE --key "{\"ticketId\":{\"S\":\"$TICKET_ID\"}}" 2>/dev/null)
    if [ $? -eq 0 ]; then
        echo -e "$AWS_RECORD"
    else
        echo -e "${RED}Failed to retrieve record from DynamoDB${NC}"
    fi
fi

echo -e "\n${GREEN}Deployment testing completed successfully!${NC}" 

# Check for exit a plate that is not in the system
echo -e "\n${BLUE}Checking for exit a plate that is not in the system...${NC}"
EXIT_RESPONSE=$(curl -s -X POST "https://$API_URL/exit?ticketId=NONEXISTENT")
echo "Response: $EXIT_RESPONSE"
