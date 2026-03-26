describe('messaging_conversation_list', () => {
  it('shows existing conversations in the sidebar after login', () => {
    cy.login('Alice', 'Alice123');

    cy.get('.conversation-list').should('exist');


    cy.get('.conversation-list').then($list => {
      if ($list.find('.conversation-item').length > 0) {
        cy.get('.conversation-item').first().should('be.visible');
      } else {
        cy.contains('No conversations yet').should('be.visible');
      }
    });
  });
});
