# Stripe Integration Setup Guide

## 🎯 **Overview**
This guide configures Stripe payment processing with Supabase using the Stripe Wrapper integration for subscription management in the DFS Lineup Optimizer.

## 📋 **Prerequisites**
- Supabase project with schema deployed
- Stripe account with API keys
- Auth migration completed (users linked to auth.users)

## 🔧 **Step 1: Supabase Stripe Wrapper Configuration**

### 1.1 Access Stripe Wrapper
1. Go to **Supabase Dashboard** > **Integrations**
2. Find **"Stripe Wrapper"** in the list
3. Click **"Add new wrapper"**

### 1.2 Configure Wrapper Settings
```sql
-- Wrapper Configuration (run in Supabase SQL Editor)
CREATE FOREIGN DATA WRAPPER stripe_wrapper
  HANDLER stripe_fdw_handler
  VALIDATOR stripe_fdw_validator;

-- Create Stripe server
CREATE SERVER stripe_server
  FOREIGN DATA WRAPPER stripe_wrapper
  OPTIONS (
    api_url 'https://api.stripe.com/v1'
  );

-- Create user mapping (use your Stripe secret key)
CREATE USER MAPPING FOR postgres
  SERVER stripe_server
  OPTIONS (
    api_key 'sk_test_your_stripe_secret_key_here'  -- Replace with actual key
  );
```

### 1.3 Set Up Stripe Tables
```sql
-- Import Stripe customers table
IMPORT FOREIGN SCHEMA stripe
  LIMIT TO (customers, subscriptions, invoices, payment_methods, prices, products)
  FROM SERVER stripe_server 
  INTO stripe_api;

-- Create local Stripe sync tables (already in schema)
-- These tables sync with Stripe via webhooks
SELECT 'Stripe tables ready in public schema' AS status;
```

## 🔑 **Step 2: Environment Variables**

### 2.1 Supabase Environment Variables
Add these to your Supabase project settings:
```env
STRIPE_SECRET_KEY=sk_test_your_stripe_secret_key
STRIPE_PUBLISHABLE_KEY=pk_test_your_stripe_publishable_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret
```

### 2.2 Service Environment Variables
Update your microservices `.env` files:

**User Service (.env)**
```env
# Existing vars...
STRIPE_SECRET_KEY=sk_test_your_stripe_secret_key
STRIPE_PUBLISHABLE_KEY=pk_test_your_stripe_publishable_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret
STRIPE_WEBHOOK_ENDPOINT=https://your-domain.com/api/v1/webhooks/stripe
```

**Frontend (.env.development)**
```env
# Existing vars...
VITE_STRIPE_PUBLISHABLE_KEY=pk_test_your_stripe_publishable_key
```

## 🪝 **Step 3: Stripe Webhook Configuration**

### 3.1 Create Webhook in Stripe Dashboard
1. Go to **Stripe Dashboard** > **Developers** > **Webhooks**
2. Click **"Add endpoint"**
3. Set URL: `https://your-domain.com/api/v1/webhooks/stripe`
4. Select events:
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.payment_succeeded`
   - `invoice.payment_failed`

### 3.2 Webhook Handler (User Service)
The webhook endpoint should be added to your user service:

```go
// services/user-service/internal/api/handlers/stripe_webhooks.go
package handlers

import (
    "encoding/json"
    "io/ioutil"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/stripe/stripe-go/v72/webhook"
)

func (h *UserHandler) HandleStripeWebhook(c *gin.Context) {
    payload, err := ioutil.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
        return
    }

    signature := c.GetHeader("Stripe-Signature")
    event, err := webhook.ConstructEvent(payload, signature, h.stripeWebhookSecret)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
        return
    }

    // Handle different event types
    switch event.Type {
    case "customer.subscription.created":
        // Handle subscription creation
    case "customer.subscription.updated":
        // Handle subscription updates
    case "customer.subscription.deleted":
        // Handle subscription cancellation
    case "invoice.payment_succeeded":
        // Handle successful payment
    case "invoice.payment_failed":
        // Handle failed payment
    }

    c.JSON(http.StatusOK, gin.H{"received": true})
}
```

## 💰 **Step 4: Subscription Tier Setup**

### 4.1 Create Stripe Products & Prices
Run this in Stripe Dashboard or via API:

```javascript
// Create products and prices in Stripe
const products = [
  {
    name: 'Basic Plan',
    description: 'Basic DFS optimization features',
    id: 'prod_basic'
  },
  {
    name: 'Premium Plan', 
    description: 'Advanced DFS features with unlimited optimizations',
    id: 'prod_premium'
  }
];

const prices = [
  {
    product: 'prod_basic',
    unit_amount: 999, // $9.99
    currency: 'usd',
    recurring: { interval: 'month' },
    id: 'price_basic_monthly'
  },
  {
    product: 'prod_premium',
    unit_amount: 2999, // $29.99
    currency: 'usd', 
    recurring: { interval: 'month' },
    id: 'price_premium_monthly'
  }
];
```

### 4.2 Update Subscription Tiers in Database
```sql
-- Update subscription tiers with Stripe IDs
UPDATE public.subscription_tiers 
SET 
    stripe_price_id = CASE 
        WHEN name = 'basic' THEN 'price_basic_monthly'
        WHEN name = 'premium' THEN 'price_premium_monthly' 
        ELSE NULL 
    END,
    stripe_product_id = CASE
        WHEN name = 'basic' THEN 'prod_basic'
        WHEN name = 'premium' THEN 'prod_premium'
        ELSE NULL
    END
WHERE name IN ('basic', 'premium');

-- Verify the update
SELECT name, stripe_product_id, stripe_price_id, price_cents 
FROM public.subscription_tiers;
```

## 🧪 **Step 5: Testing Stripe Integration**

### 5.1 Test Customer Creation
```sql
-- Test creating a Stripe customer for existing user
SELECT 'Testing Stripe customer creation...' AS test;

-- This should be done via your user service API
-- Example API call to create customer:
-- POST /api/v1/users/:id/stripe-customer
```

### 5.2 Test Subscription Flow
```javascript
// Frontend test - create subscription
const handleSubscribe = async (priceId) => {
  try {
    const response = await fetch('/api/v1/subscriptions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({ price_id: priceId })
    });
    
    const { client_secret } = await response.json();
    
    // Redirect to Stripe Checkout or use Payment Element
    const stripe = await stripePromise;
    await stripe.redirectToCheckout({
      sessionId: client_secret
    });
  } catch (error) {
    console.error('Subscription error:', error);
  }
};
```

## 🔍 **Step 6: Verification Checklist**

### ✅ **Stripe Wrapper Status**
- [ ] Wrapper configured in Supabase
- [ ] Foreign tables accessible
- [ ] API key properly set

### ✅ **Database Integration**
- [ ] Stripe tables exist (customers, subscriptions, events)
- [ ] Subscription tiers have Stripe IDs
- [ ] Webhooks table ready for events

### ✅ **Service Integration**
- [ ] User service has Stripe client
- [ ] Webhook endpoints configured
- [ ] Environment variables set

### ✅ **Frontend Integration**
- [ ] Stripe.js loaded
- [ ] Publishable key configured
- [ ] Subscription UI ready

## 🚨 **Security Notes**

1. **Never expose secret keys** in frontend code
2. **Use webhooks** for reliable subscription status updates
3. **Validate webhook signatures** to prevent fraud
4. **Use test keys** during development
5. **Enable Strong Customer Authentication** (SCA) for EU compliance

## 🔧 **Troubleshooting**

**Error: "Invalid API key"**
- Verify Stripe secret key in environment variables
- Check if using test vs live keys consistently

**Error: "Webhook signature verification failed"**
- Ensure webhook secret matches Stripe dashboard
- Check request headers are properly forwarded

**Error: "Customer not found"**
- Verify customer creation in webhook handlers
- Check stripe_customer_id in users table

## 📚 **Next Steps**

After completing this setup:
1. Update user service with Stripe customer creation
2. Add subscription management endpoints
3. Implement usage tracking and limits
4. Test end-to-end payment flow
5. Configure production webhooks and keys