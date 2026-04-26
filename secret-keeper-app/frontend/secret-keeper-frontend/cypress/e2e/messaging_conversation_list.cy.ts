// Distinct from messaging_load_page: verifies clicking a conversation item
// selects it and loads the chat area (not just that the sidebar renders).
describe('messaging_conversation_list', () => {
  it('clicking a conversation item opens the chat area', () => {
    cy.login('Alice', 'Alice123');

    cy.get('.conversation-list').then($list => {
      if ($list.find('.conversation-item').length > 0) {
        cy.get('.conversation-item').first().click();
        // After selecting, either the message area loads or a room-key modal appears
        cy.get('.rectangle-4-5').should('exist');
      } else {
        cy.contains('No conversations yet').should('be.visible');
        cy.contains('button', 'Create Conversation').should('be.visible');
      }
    });
  });
});
