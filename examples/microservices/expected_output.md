# Microservices Platform Context

Aggregated architecture and service documentation.

## Table of Contents

* [Common Packages](#common-packages)
  * [Shared Utilities](#shared-utilities)
    * [Resilience](#resilience)
      * [Exponential Backoff](#exponential-backoff)
* [Core Services](#core-services)
  * [Auth Service](#auth-service)
    * [Auth Service](#auth-service-1)
      * [Endpoints](#endpoints)
        * [POST /api/v1/login](#post-apiv1login)
  * [Payment Service](#payment-service)
    * [Payment Service](#payment-service-1)
      * [Webhooks](#webhooks)
        * [POST /webhooks/stripe](#post-webhooksstripe)

---

## Common Packages

> *Source: `packages/common/docs/utils.md`*

| Property | Value |
| :--- | :--- |
| **package** | @org/common |
| **title** | Common Utilities |
| **version** | 2.1.0 |

### Shared Utilities

Standard helper functions used across microservices.

#### Resilience

##### Exponential Backoff

All network calls use jittered backoff.

## Core Services

### Auth Service

> *Source: `services/auth/docs/overview.md`*

| Property | Value |
| :--- | :--- |
| **port** | 8081 |
| **service** | auth-svc |
| **title** | Authentication Service Overview |

#### Auth Service

Handles user authentication and OAuth2 token issuance.

##### Endpoints

###### POST /api/v1/login

Authenticates credentials and returns JWT.

### Payment Service

> *Source: `services/payment/docs/api.md`*

| Property | Value |
| :--- | :--- |
| **port** | 8082 |
| **service** | payment-svc |
| **title** | Payment Service API |

#### Payment Service

Integrates with Stripe and PayPal for payment settlement.

##### Webhooks

###### POST /webhooks/stripe

Handles Stripe charge succeeded events.

