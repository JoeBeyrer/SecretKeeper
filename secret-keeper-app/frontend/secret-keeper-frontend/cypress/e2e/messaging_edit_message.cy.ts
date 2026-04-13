describe('messaging_edit_message', () => {
  it('lets a user edit a sent message from the message actions menu', () => {
    cy.login('Alice', 'Alice123');

    cy.get('input[placeholder="Username to chat with"]').clear().type('Bob');
    cy.contains('button', 'New').click();

    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').then(($title) => {
      const title = $title.text().trim();
      if (title === 'Choose Room Key') {
        cy.contains('button', 'Create Conversation').click();
        cy.contains('.modal-title', 'Your Room Key').should('be.visible');
        cy.contains('button', "I've saved it").click();
      } else if (title === 'Your Room Key') {
        cy.contains('button', "I've saved it").click();
      }
    });

    cy.get('.message-input-bar input[placeholder="Type your message here..."]').type('edit me');
    cy.get('.message-input-bar .rectangle-1-3').click();

    cy.contains('.message-bubble.sent', 'edit me').should('be.visible');
    cy.get('.message-row.mine .message-menu-trigger', { timeout: 10000 }).should('not.be.disabled').click();
    cy.contains('.message-menu-item', 'Edit').click();

    cy.get('.message-edit-input').should('be.visible').clear().type('edited message');
    cy.get('.message-edit-confirm').click();

    cy.contains('.message-bubble.sent', 'edited message', { timeout: 10000 }).should('be.visible');
  });
});
