# Supabase Setup Guide for DFS Optimizer

## Phone Authentication Configuration

### 1. Enable Phone Auth in Dashboard
1. Go to **Authentication > Providers** in your Supabase dashboard
2. Find **Phone** provider and click **Enable**
3. Configure the following settings:

### 2. SMS Provider Setup (Choose one)

#### Option A: Twilio (Recommended for Production)
```
Provider: Twilio
Account SID: your_twilio_account_sid
Auth Token: your_twilio_auth_token
From Number: +1234567890 (your Twilio phone number)
```

#### Option B: MessageBird
```
Provider: MessageBird
Access Key: your_messagebird_access_key
From Number: +1234567890
```

#### Option C: Textlocal
```
Provider: Textlocal
API Key: your_textlocal_api_key
Sender: your_sender_name
```

### 3. Phone Auth Settings
```
Enable phone confirmations: ✓ Enabled
Phone confirmation timeout: 300 seconds (5 minutes)
Phone confirmation redirect URL: https://your-app.com/auth/callback
```

## Database Migration Steps

### 1. Access SQL Editor
1. Go to **SQL Editor** in your Supabase dashboard
2. Click **New Query**
3. Copy and paste the migration script below

### 2. Run the Migration
Execute the following SQL script in the SQL Editor:

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable realtime for user data
ALTER PUBLICATION supabase_realtime ADD TABLE users;
ALTER PUBLICATION supabase_realtime ADD TABLE user_preferences;

-- Users table (extends auth.users)
CREATE TABLE public.users (
  id UUID REFERENCES auth.users(id) PRIMARY KEY,
  phone_number TEXT UNIQUE NOT NULL,
  first_name TEXT,
  last_name TEXT,
  subscription_tier TEXT DEFAULT 'free',
  subscription_status TEXT DEFAULT 'active',
  subscription_expires_at TIMESTAMPTZ,
  stripe_customer_id TEXT,
  monthly_optimizations_used INTEGER DEFAULT 0,
  monthly_simulations_used INTEGER DEFAULT 0,
  usage_reset_date DATE DEFAULT CURRENT_DATE,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Continue with rest of migration...
```

### 3. Verify Migration
After running the migration, verify the tables were created:
```sql
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN ('users', 'user_preferences', 'subscription_tiers', 'legacy_user_mapping');
```

You should see 4 tables returned.

## Environment Variable Configuration

### Backend (.env)
```bash
# Supabase Configuration
SUPABASE_URL=https://your-project-id.supabase.co
SUPABASE_SERVICE_KEY=your_service_role_key
SUPABASE_ANON_KEY=your_anon_key

# Database (Supabase provides this)
DATABASE_URL=postgresql://postgres:[password]@db.[project-id].supabase.co:5432/postgres

# Existing configuration
JWT_SECRET=your_existing_jwt_secret
PORT=8080
CORS_ORIGINS=http://localhost:3000,http://localhost:5173
```

### Frontend (.env)
```bash
# Supabase Configuration
VITE_SUPABASE_URL=https://your-project-id.supabase.co
VITE_SUPABASE_ANON_KEY=your_anon_key

# API Configuration
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Testing the Setup

Run the test script to validate everything is working:
```bash
chmod +x test-supabase-migration.sh
./test-supabase-migration.sh
```

## Common Issues & Solutions

### Issue: "Failed to create user profile"
**Solution:** Check that the `users` table exists and RLS policies allow inserts

### Issue: "JWT validation failed"
**Solution:** Verify your service role key is correct and has proper permissions

### Issue: "Real-time not working"
**Solution:** Ensure tables are added to the realtime publication:
```sql
ALTER PUBLICATION supabase_realtime ADD TABLE users;
ALTER PUBLICATION supabase_realtime ADD TABLE user_preferences;
```

### Issue: "Phone auth not sending SMS"
**Solution:** 
1. Check SMS provider configuration in Auth > Providers
2. Verify you have credits/quota with your SMS provider
3. Check phone number format (must be E.164: +1234567890)

## Next Steps After Setup

1. **Test phone authentication** with real phone number
2. **Deploy to staging** with production Supabase project
3. **Set up monitoring** for real-time connections
4. **Configure backup strategy** for user data
5. **Set up production SMS provider** with sufficient quota

## Security Checklist

- [ ] Service role key is not exposed in frontend code
- [ ] RLS policies are enabled on all user tables
- [ ] Real-time access is properly restricted
- [ ] SMS provider has rate limiting configured
- [ ] Database password is strong and secure
- [ ] API keys are stored securely in environment variables