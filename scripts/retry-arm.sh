#!/bin/bash
# Retry creating an ARM instance until capacity is available.
# Run: bash scripts/retry-arm.sh
# Stop: Ctrl+C
#
# When ARM succeeds, you can terminate the x86 micro and switch to the ARM instance.
# Update SUBNET_ID if you created a new subnet, or reuse the existing one.

TENANCY="ocid1.tenancy.oc1..aaaaaaaah4fr4536wp5ffjk3otqi5b7phqgtj6sxy2wjgips4byhm3sjdbtq"
SUBNET_ID="ocid1.subnet.oc1.iad.aaaaaaaaeqy5amzabr5klquvke4gvyl6ujdkipofo2dk6lqqmtsorchjsfoa"
IMAGE_ID="ocid1.image.oc1.iad.aaaaaaaaccnswiekwi4w3pkmygjvfk24epduwj7uvq2smjmznu4kq6dcs27a"
SSH_KEY_FILE="d:/repos/tewodros-terminal/.ssh/oracle_vm.pub"
ADS=("mqKE:US-ASHBURN-AD-1" "mqKE:US-ASHBURN-AD-2" "mqKE:US-ASHBURN-AD-3")

INTERVAL=300  # seconds between retry rounds (5 minutes)
MAX_HOURS=48  # give up after this many hours

echo "=== ARM Instance Retry Script ==="
echo "Retrying every ${INTERVAL}s across ${#ADS[@]} ADs"
echo "Will run for up to ${MAX_HOURS} hours. Ctrl+C to stop."
echo ""

START=$(date +%s)
ATTEMPT=0

while true; do
    ELAPSED=$(( ($(date +%s) - START) / 3600 ))
    if [ "$ELAPSED" -ge "$MAX_HOURS" ]; then
        echo "[$(date)] Gave up after ${MAX_HOURS} hours."
        exit 1
    fi

    ATTEMPT=$((ATTEMPT + 1))
    echo "[$(date)] Round $ATTEMPT (${ELAPSED}h elapsed)"

    for AD in "${ADS[@]}"; do
        echo "  Trying $AD..."
        RESULT=$(oci compute instance launch --auth security_token \
            --compartment-id "$TENANCY" \
            --availability-domain "$AD" \
            --display-name "tewodros-terminal-arm" \
            --shape "VM.Standard.A1.Flex" \
            --shape-config '{"ocpus":1,"memoryInGBs":6}' \
            --image-id "$IMAGE_ID" \
            --subnet-id "$SUBNET_ID" \
            --assign-public-ip true \
            --ssh-authorized-keys-file "$SSH_KEY_FILE" 2>&1)

        if echo "$RESULT" | grep -q '"lifecycle-state"'; then
            echo ""
            echo "========================================="
            echo "  SUCCESS! ARM instance created in $AD"
            echo "========================================="
            echo "$RESULT"
            echo ""
            echo "Next steps:"
            echo "  1. Get the public IP from the output above"
            echo "  2. Update your deploy config and Cloudflare DNS"
            echo "  3. Terminate the x86 micro instance"
            exit 0
        fi
    done

    echo "  No capacity. Sleeping ${INTERVAL}s..."
    sleep "$INTERVAL"
done
