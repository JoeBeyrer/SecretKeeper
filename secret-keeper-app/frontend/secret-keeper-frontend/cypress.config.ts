import { defineConfig } from 'cypress'
import { execSync } from 'child_process'

const DB_PATH = '/Users/christianfarese/Projects/SecretKeeper/secret-keeper-app/backend/database/secretkeeper.db'

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:4200',
    setupNodeEvents(on) {
      on('task', {
        clearDB() {
          execSync(`sqlite3 "${DB_PATH}" "
            DELETE FROM message_reactions;
            DELETE FROM messages;
            DELETE FROM conversation_pending_room_keys;
            DELETE FROM conversation_keys;
            DELETE FROM conversation_members;
            DELETE FROM conversations;
            DELETE FROM friendships;
            DELETE FROM blocks;
            DELETE FROM sessions;
            DELETE FROM password_resets;
            DELETE FROM password_reset_audit;
            DELETE FROM email_verifications;
          "`);
          return null;
        }
      });
    }
  },
  component: {
    devServer: {
      framework: 'angular',
      bundler: 'webpack',
    },
    specPattern: '**/*.cy.ts'
  }
})
