---
title: "Payment Service API"
service: "payment-svc"
port: 8082
---
# Payment Service

Integrates with Stripe and PayPal for payment settlement.

## Webhooks

### POST /webhooks/stripe

Handles Stripe charge succeeded events.
