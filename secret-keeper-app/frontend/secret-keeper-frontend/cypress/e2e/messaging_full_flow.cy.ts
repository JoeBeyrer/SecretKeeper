// All tests that require an open Alice→Bob conversation live here.
// The before hook clears the DB, logs in as Alice, and creates the
// conversation exactly once. Every it block then runs sequentially
// in that same browser session so the room key is never lost.
describe('messaging_full_flow', { testIsolation: false }, () => {
  before(() => {
    cy.task('clearDB');
    cy.login('Alice', 'Alice123');

    cy.visit('/messaging?chatWith=Bob');
    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').should('contain', 'Choose Room Key');
    cy.contains('button', 'Create Conversation').click();
    cy.contains('.modal-title', 'Your Room Key').should('be.visible');
    cy.contains('button', "I've saved it").click();
    cy.get('.modal').should('not.exist');
  });

  it('shows the new conversation in the sidebar list', () => {
    cy.get('.conversation-item').should('have.length.at.least', 1);
    cy.get('.rectangle-4-5').should('exist');
  });

  it('sends and edits a message', () => {
    cy.get('.message-input-bar input[placeholder="Type your message here..."]').type('edit me');
    cy.get('.message-input-bar .rectangle-1-3').click();
    cy.contains('.message-bubble.sent', 'edit me').should('be.visible');

    cy.get('.message-row.mine .message-menu-trigger', { timeout: 10000 })
      .should('not.be.disabled')
      .first()
      .click();
    cy.contains('.message-menu-item', 'Edit').click();
    cy.get('.message-edit-input').should('be.visible').clear().type('edited message');
    cy.get('.message-edit-confirm').click();
    cy.contains('.message-bubble.sent', 'edited message', { timeout: 10000 }).should('be.visible');
  });

  it('adds an emoji reaction to a message', () => {
    cy.get('.message-input-bar input[placeholder="Type your message here..."]').type('react to this');
    cy.get('.message-input-bar .rectangle-1-3').click();
    cy.contains('.message-bubble.sent', 'react to this', { timeout: 10000 }).should('be.visible');

    cy.get('.message-row.mine .emoji-react-btn', { timeout: 10000 })
      .should('not.be.disabled')
      .last()
      .click();
    cy.get('.emoji-picker-popup').should('be.visible');
    cy.get('.emoji-option').contains('👍').click();
    cy.get('.reaction-chip').contains('👍').should('be.visible');
  });

  it('sends a file attachment and shows it in the chat', () => {
    cy.get('.hidden-attachment-input').selectFile('cypress/fixtures/sample-upload.txt', { force: true });
    cy.contains('.pending-attachment-chip', 'sample-upload.txt').should('be.visible');
    cy.get('.message-input-bar .rectangle-1-3').click();
    cy.get('.pending-attachments-bar').should('not.exist');
    cy.contains('.message-attachment-name', 'sample-upload.txt', { timeout: 10000 }).should('be.visible');
    cy.contains('.message-attachment-download', 'Download').should('be.visible');
  });

  it('deletes a sent message', () => {
    cy.get('.message-input-bar input[placeholder="Type your message here..."]').type('delete me');
    cy.get('.message-input-bar .rectangle-1-3').click();
    cy.contains('.message-bubble.sent', 'delete me', { timeout: 10000 }).should('be.visible');

    cy.get('.message-row.mine .delete-msg-btn', { timeout: 10000 })
      .should('not.be.disabled')
      .last()
      .click();
    cy.contains('.message-bubble.sent', 'delete me').should('not.exist');
  });

  it('sends a searchable message', () => {
    cy.get('.message-input-bar input[placeholder="Type your message here..."]').type('searchable content xyz');
    cy.get('.message-input-bar .rectangle-1-3').click();
    cy.contains('.message-bubble.sent', 'searchable content xyz', { timeout: 10000 }).should('be.visible');
  });

  it('opens the search bar when the search icon is clicked', () => {
    cy.get('.search-toggle-btn').click();
    cy.get('.msg-search-bar').should('be.visible');
    cy.get('.msg-search-input').should('be.visible');
  });

  it('filters messages and highlights matches', () => {
    cy.get('.msg-search-input').type('searchable');
    cy.get('.msg-search-count').should('contain', '1 result');
    cy.get('.msg-highlight').should('be.visible').and('contain', 'searchable');
  });

  it('shows zero results for a non-matching query', () => {
    cy.get('.msg-search-input').clear().type('zzznomatch999');
    cy.get('.msg-search-count').should('contain', '0 results');
    cy.get('.messages-area .message-row').should('not.exist');
  });

  it('closes the search bar and restores all messages', () => {
    cy.get('.search-toggle-btn').click();
    cy.get('.msg-search-bar').should('not.exist');
    cy.contains('.message-bubble.sent', 'searchable content xyz').should('be.visible');
  });
});
