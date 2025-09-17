# User API Module

This module provides comprehensive user management APIs including authentication, profile management, security settings, and third-party integrations.

## API Endpoints

### Authentication

| Method | Endpoint         | Auth     | Description                  |
| ------ | ---------------- | -------- | ---------------------------- |
| GET    | `/user/login`    | Public   | Get login page configuration |
| POST   | `/user/login`    | Public   | User login                   |
| POST   | `/user/register` | Public   | User registration            |
| POST   | `/user/logout`   | Required | User logout                  |

### Profile Management

| Method | Endpoint        | Auth     | Description         |
| ------ | --------------- | -------- | ------------------- |
| GET    | `/user/profile` | Required | Get user profile    |
| PUT    | `/user/profile` | Required | Update user profile |

### Account Security

| Method | Endpoint                                 | Auth     | Description                                        |
| ------ | ---------------------------------------- | -------- | -------------------------------------------------- |
| PUT    | `/user/account/password`                 | Required | Change password (requires current password or 2FA) |
| POST   | `/user/account/password/reset/request`   | Public   | Request password reset (rate-limited)              |
| POST   | `/user/account/password/reset/verify`    | Public   | Verify reset token and set new password            |
| GET    | `/user/account/email`                    | Required | Get current email info                             |
| POST   | `/user/account/email/change/request`     | Required | Request email change (sends code to current email) |
| POST   | `/user/account/email/change/verify`      | Required | Verify email change with code                      |
| POST   | `/user/account/email/verification-code`  | Required | Send verification code to current email            |
| POST   | `/user/account/email/verify`             | Required | Verify current email                               |
| GET    | `/user/account/mobile`                   | Required | Get current mobile info                            |
| POST   | `/user/account/mobile/change/request`    | Required | Request mobile change                              |
| POST   | `/user/account/mobile/change/verify`     | Required | Verify mobile change with code                     |
| POST   | `/user/account/mobile/verification-code` | Required | Send verification code to mobile                   |
| POST   | `/user/account/mobile/verify`            | Required | Verify current mobile                              |

### Multi-Factor Authentication (MFA)

| Method | Endpoint                                   | Auth     | Description                              |
| ------ | ------------------------------------------ | -------- | ---------------------------------------- |
| GET    | `/user/mfa/totp`                           | Required | Get TOTP QR code and setup info          |
| POST   | `/user/mfa/totp/enable`                    | Required | Enable TOTP with verification            |
| POST   | `/user/mfa/totp/disable`                   | Required | Disable TOTP with verification           |
| POST   | `/user/mfa/totp/verify`                    | Required | Verify TOTP code                         |
| GET    | `/user/mfa/totp/recovery-codes`            | Required | Get TOTP recovery codes                  |
| POST   | `/user/mfa/totp/recovery-codes/regenerate` | Required | Regenerate recovery codes                |
| POST   | `/user/mfa/totp/reset`                     | Required | Reset TOTP (requires email verification) |
| GET    | `/user/mfa/sms`                            | Required | Get SMS MFA status                       |
| POST   | `/user/mfa/sms/enable`                     | Required | Enable SMS MFA                           |
| POST   | `/user/mfa/sms/disable`                    | Required | Disable SMS MFA                          |
| POST   | `/user/mfa/sms/verification-code`          | Required | Send SMS verification code               |
| POST   | `/user/mfa/sms/verify`                     | Required | Verify SMS code                          |

### OAuth & Third-Party Integration

| Method | Endpoint                                  | Auth     | Description                          |
| ------ | ----------------------------------------- | -------- | ------------------------------------ |
| GET    | `/user/oauth/providers`                   | Required | Get linked OAuth providers           |
| DELETE | `/user/oauth/:provider`                   | Required | Unlink OAuth provider                |
| GET    | `/user/oauth/providers/available`         | Public   | Get available OAuth providers        |
| GET    | `/user/oauth/:provider/authorize`         | Public   | Get OAuth authorization URL          |
| POST   | `/user/oauth/:provider/connect`           | Required | Connect OAuth provider               |
| POST   | `/user/oauth/:provider/authorize/prepare` | Public   | Handle POST callback (Apple, WeChat) |
| POST   | `/user/oauth/:provider/callback`          | Public   | Handle GET callback (Google, GitHub) |

### API Keys Management

| Method | Endpoint                            | Auth     | Description                        |
| ------ | ----------------------------------- | -------- | ---------------------------------- |
| GET    | `/user/api-keys`                    | Required | Get all user API keys              |
| POST   | `/user/api-keys`                    | Required | Create new API key                 |
| GET    | `/user/api-keys/:key_id`            | Required | Get specific API key details       |
| PUT    | `/user/api-keys/:key_id`            | Required | Update API key (name, permissions) |
| DELETE | `/user/api-keys/:key_id`            | Required | Delete API key                     |
| POST   | `/user/api-keys/:key_id/regenerate` | Required | Regenerate API key                 |

### Credits & Top-up

| Method | Endpoint                        | Auth     | Description                |
| ------ | ------------------------------- | -------- | -------------------------- |
| GET    | `/user/credits`                 | Required | Get user credits info      |
| GET    | `/user/credits/history`         | Required | Get credits change history |
| GET    | `/user/credits/topup`           | Required | Get topup records          |
| POST   | `/user/credits/topup`           | Required | Create topup order         |
| GET    | `/user/credits/topup/:order_id` | Required | Get topup order status     |
| POST   | `/user/credits/topup/card-code` | Required | Redeem card code           |

### Subscription Management

| Method | Endpoint             | Auth     | Description              |
| ------ | -------------------- | -------- | ------------------------ |
| GET    | `/user/subscription` | Required | Get user subscription    |
| PUT    | `/user/subscription` | Required | Update user subscription |

### Usage Statistics

| Method | Endpoint                 | Auth     | Description               |
| ------ | ------------------------ | -------- | ------------------------- |
| GET    | `/user/usage/statistics` | Required | Get user usage statistics |
| GET    | `/user/usage/history`    | Required | Get user usage history    |

### Billing & Invoices

| Method | Endpoint                 | Auth     | Description              |
| ------ | ------------------------ | -------- | ------------------------ |
| GET    | `/user/billing/history`  | Required | Get user billing history |
| GET    | `/user/billing/invoices` | Required | Get user invoices list   |

### Referral & Invitations

| Method | Endpoint                     | Auth     | Description                   |
| ------ | ---------------------------- | -------- | ----------------------------- |
| GET    | `/user/referral/code`        | Required | Get user referral code        |
| GET    | `/user/referral/statistics`  | Required | Get user referral statistics  |
| GET    | `/user/referral/history`     | Required | Get user referral history     |
| GET    | `/user/referral/commissions` | Required | Get user referral commissions |

### Team Management

#### Team CRUD

| Method | Endpoint               | Auth     | Description           |
| ------ | ---------------------- | -------- | --------------------- |
| GET    | `/user/teams`          | Required | Get user teams        |
| POST   | `/user/teams`          | Required | Create user team      |
| GET    | `/user/teams/:team_id` | Required | Get user team details |
| PUT    | `/user/teams/:team_id` | Required | Update user team      |
| DELETE | `/user/teams/:team_id` | Required | Delete user team      |

#### Member Management

| Method | Endpoint                                  | Auth     | Description                       |
| ------ | ----------------------------------------- | -------- | --------------------------------- |
| GET    | `/user/teams/:team_id/members`            | Required | Get user team members             |
| GET    | `/user/teams/:team_id/members/:member_id` | Required | Get user team member details      |
| POST   | `/user/teams/:team_id/members/direct`     | Required | Add member directly (bots/system) |
| PUT    | `/user/teams/:team_id/members/:member_id` | Required | Update user team member           |
| DELETE | `/user/teams/:team_id/members/:member_id` | Required | Remove user team member           |

#### Team Invitations

| Method | Endpoint                                                 | Auth     | Description            |
| ------ | -------------------------------------------------------- | -------- | ---------------------- |
| POST   | `/user/teams/:team_id/invitations`                       | Required | Send team invitation   |
| GET    | `/user/teams/:team_id/invitations`                       | Required | Get team invitations   |
| GET    | `/user/teams/:team_id/invitations/:invitation_id`        | Required | Get invitation details |
| PUT    | `/user/teams/:team_id/invitations/:invitation_id/resend` | Required | Resend invitation      |
| DELETE | `/user/teams/:team_id/invitations/:invitation_id`        | Required | Cancel invitation      |

### Invitation Response (Cross-module)

_Universal invitation response endpoints that handle invitations from any module (teams, organizations, etc.)_

| Method | Endpoint                           | Auth     | Description                  |
| ------ | ---------------------------------- | -------- | ---------------------------- |
| GET    | `/user/invitations/:token`         | Public   | Get invitation info by token |
| POST   | `/user/invitations/:token/accept`  | Required | Accept invitation            |
| POST   | `/user/invitations/:token/decline` | Public   | Decline invitation           |

### User Preferences

| Method | Endpoint                   | Auth     | Description                 |
| ------ | -------------------------- | -------- | --------------------------- |
| GET    | `/user/preferences`        | Required | Get user preferences        |
| GET    | `/user/preferences/schema` | Required | Get user preferences schema |
| PUT    | `/user/preferences`        | Required | Update user preferences     |

### Privacy Settings

| Method | Endpoint               | Auth     | Description                  |
| ------ | ---------------------- | -------- | ---------------------------- |
| GET    | `/user/privacy`        | Required | Get user privacy settings    |
| GET    | `/user/privacy/schema` | Required | Get user privacy schema      |
| PUT    | `/user/privacy`        | Required | Update user privacy settings |

### User Management (Admin)

| Method | Endpoint               | Auth     | Description      |
| ------ | ---------------------- | -------- | ---------------- |
| GET    | `/user/users`          | Required | Get users        |
| POST   | `/user/users`          | Required | Create user      |
| GET    | `/user/users/:user_id` | Required | Get user details |
| PUT    | `/user/users/:user_id` | Required | Update user      |
| DELETE | `/user/users/:user_id` | Required | Delete user      |

## Authentication

- **Public**: No authentication required
- **Required**: Requires valid OAuth token via `oauth.Guard` middleware

## Notes

- All endpoints return JSON responses
- Rate limiting may apply to sensitive operations (password reset, verification codes)
- This module is designed to eventually replace the `signin` module
- OAuth callbacks support both GET (Google, GitHub) and POST (Apple, WeChat) methods

## Architecture

### Modular Design

- **Team Management**: Handles team CRUD, member management, and invitation sending
- **Invitation Response**: Universal cross-module invitation handling (accept/decline)
- **Dual Member Addition**: Supports both direct addition (bots/system) and invitation flow (users)

### Invitation Flow

1. **Send Invitation**: `POST /user/teams/:team_id/invitations`
2. **Manage Invitations**: View, resend, or cancel via team-specific endpoints
3. **Respond to Invitation**: Universal endpoints handle acceptance/decline regardless of source module
