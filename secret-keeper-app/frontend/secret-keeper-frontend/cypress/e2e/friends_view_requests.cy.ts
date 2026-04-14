describe('friends_view_requests', () => {
  it('switches to the Requests tab and displays content', () => {
    cy.login('Bob', 'Bob12345');
    cy.visit('/friends');

    cy.contains('.tab', 'Requests').click();

    cy.get('.panel-body').should('exist').then($body => {
      if ($body.find('.friend-row').length > 0) {
        cy.get('.friend-row').first().should('be.visible');
      } else {
        cy.contains('No pending requests').should('be.visible');
      }
    });
  });
});
