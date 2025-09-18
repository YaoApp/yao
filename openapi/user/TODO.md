# User Module TODO

## ✅ Implemented (5/80)

### Authentication

- ✅ GET `/user/login` - Get login page configuration
- ✅ POST `/user/login` - User login

### OAuth & Third-Party Integration

- ✅ GET `/user/oauth/:provider/authorize` - Get OAuth authorization URL
- ✅ POST `/user/oauth/:provider/authorize/prepare` - Handle OAuth POST callback (Apple, WeChat)
- ✅ POST `/user/oauth/:provider/callback` - Handle OAuth GET callback (Google, GitHub)

## ❌ TODO (75/80)

### Authentication

- ❌ POST `/user/register` - User registration
- ❌ POST `/user/logout` - User logout

### Profile Management

- ❌ GET `/user/profile` - Get user profile
- ❌ PUT `/user/profile` - Update user profile

### Account Security (13 endpoints)

- ❌ Password management (3 endpoints)
- ❌ Email management (5 endpoints)
- ❌ Mobile management (5 endpoints)

### Multi-Factor Authentication (12 endpoints)

- ❌ TOTP management (7 endpoints)
- ❌ SMS MFA management (5 endpoints)

### OAuth & Third-Party Integration

- ❌ GET `/user/oauth/providers` - Get linked OAuth providers
- ❌ DELETE `/user/oauth/:provider` - Unlink OAuth provider
- ❌ GET `/user/oauth/providers/available` - Get available OAuth providers
- ❌ POST `/user/oauth/:provider/connect` - Connect OAuth provider

### API Keys Management (6 endpoints)

- ❌ CRUD operations and regeneration for API keys

### Credits & Top-up (6 endpoints)

- ❌ Credits info, history, and top-up management

### Subscription Management (2 endpoints)

- ❌ Subscription info and updates

### Usage Statistics (2 endpoints)

- ❌ Usage statistics and history

### Billing & Invoices (2 endpoints)

- ❌ Billing history and invoice list

### Referral & Invitations (4 endpoints)

- ❌ Referral codes, statistics, history, commissions

### Team Management (15 endpoints)

- ❌ Team CRUD (5 endpoints)
- ❌ Member management (5 endpoints)
- ❌ Invitation management (5 endpoints)

### Invitation Response (3 endpoints)

- ❌ Cross-module invitation handling

### User Preferences (3 endpoints)

- ❌ User preference settings

### Privacy Settings (3 endpoints)

- ❌ Privacy settings

### User Management (Admin) (5 endpoints)

- ❌ User CRUD operations

## Progress Summary

- **Completion**: 6.25% (5/80)
- **Core Features**: Authentication and OAuth completed
- **Next Steps**: Recommend implementing basic user management (register, logout, profile) first
