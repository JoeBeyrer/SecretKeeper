// Opens or joins a conversation with Bob via the chatWith URL param.
// Handles all three possible modal states cleanly.
function openConversationWithBob() {
  cy.visit('/messaging?chatWith=Bob');

  cy.get('.modal').should('be.visible');
  cy.get('.modal-title').then(($title) => {
    const title = $title.text().trim();
    if (title === 'Choose Room Key') {
      cy.contains('button', 'Create Conversation').click();
      cy.contains('.modal-title', 'Your Room Key').should('be.visible');
      cy.contains('button', "I've saved it").click();
    } else if (title === 'Your Room Key') {
      cy.contains('button', "I've saved it").click();
    } else {
      // Enter Room Key — rejoin existing conversation
      cy.get('.modal input[type="text"], .modal input[type="password"]').first().type('testkey123');
      cy.contains('button', 'Join').click();
    }
  });

  cy.get('.modal').should('not.exist');
}

describe('messaging_conversation_actions', () => {
  before(() => {
    cy.login('Alice', 'Alice123');
    openConversationWithBob();
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
});
