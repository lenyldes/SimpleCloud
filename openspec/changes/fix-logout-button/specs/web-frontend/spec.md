## ADDED Requirements

### Requirement: User Profile Menu and Session Logout
The web frontend SHALL provide an interactive user profile menu in the header bar displaying the authenticated user's account details and providing an explicit logout mechanism that terminates the server session, purges client-side state, and prompts for re-authentication.

#### Scenario: Opening user profile menu
- **WHEN** user clicks on the user profile container or avatar in the top header
- **THEN** system SHALL display the profile dropdown menu containing the user's email address and a logout button.

#### Scenario: Closing user profile menu on click outside
- **WHEN** the user profile dropdown menu is open and the user clicks outside the profile container
- **THEN** system SHALL close and hide the profile dropdown menu.

#### Scenario: Successful logout execution
- **WHEN** user clicks the logout button in the profile dropdown menu
- **THEN** system SHALL send a `POST /api/v1/auth/logout` request with credentials, reset client-side user and file state, hide the profile dropdown, display the authentication modal, and present a confirmation toast message.
