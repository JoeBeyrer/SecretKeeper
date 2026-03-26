describe('profile_load_page', () => {
  it('loads the profile page and displays user info', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/profile');

    cy.contains('Account Settings').should('be.visible');
    cy.get('.info-value').contains('Alice').should('be.visible');
    cy.get('.info-value').contains('Alice@secretkeeper.com').should('be.visible');
  });
});
