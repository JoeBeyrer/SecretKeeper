describe('messaging_create_conversation', () => {
  it('opens the create conversation modal and cancels', () => {
    cy.login('Alice', 'Alice123');

    cy.contains('button', 'Create Conversation').click();
    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').should('contain', 'Create Conversation');

    cy.contains('button', 'Cancel').click();
    cy.get('.modal').should('not.exist');
  });

  it('opens room key modal for Bob via chatWith param and dismisses', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/messaging?chatWith=Bob');

    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').then(($title) => {
      const title = $title.text().trim();
      if (title === 'Choose Room Key') {
        cy.contains('button', 'Cancel').click();
      } else if (title === 'Your Room Key') {
        cy.contains("I've saved it").click();
      } else if (title === 'Enter Room Key') {
        cy.contains('Cancel').click();
      }
    });

    cy.get('.modal').should('not.exist');
  });
});
